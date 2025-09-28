package app_specific

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/rs/zerolog/log"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	ectx "github.com/opisvigilant/futura/watcher/internal/ebpf/context"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -Werror" app_specific app_specific.bpf.c -- -I../../../../headers

// DatabaseMetrics represents database metrics per container
type DatabaseMetrics struct {
	QueryCount           uint64
	QueryTimeTotal       uint64
	SlowQueries          uint64
	ActiveConnections    uint64
	ConnectionTimeouts   uint64
	Transactions         uint64
	Rollbacks            uint64
	TransactionTimeTotal uint64
	LastUpdate           uint64
}

// CacheMetrics represents cache metrics per container
type CacheMetrics struct {
	CacheHits        uint64
	CacheMisses      uint64
	CacheSets        uint64
	CacheGets        uint64
	CacheDeletes     uint64
	CacheEvictions   uint64
	GetLatencyTotal  uint64
	SetLatencyTotal  uint64
	CacheSizeBytes   uint64
	LastUpdate       uint64
}

// GoMetrics represents Go runtime metrics per container
type GoMetrics struct {
	Goroutines         uint64
	GoroutineStackSize uint64
	GCCycles           uint64
	GCPauseTimeUS      uint64
	HeapSize           uint64
	HeapAlloc          uint64
	HeapIdle           uint64
	ChannelSends       uint64
	ChannelReceives    uint64
	BlockedChannels    uint64
	LastUpdate         uint64
}

// AppEvent represents a real-time application event
type AppEvent struct {
	Timestamp  uint64
	CgroupID   uint64
	PID        uint32
	EventType  uint32
	DurationUS uint64
	Value      uint64
	Details    [256]byte
}

// UprobeTarget represents a target for dynamic uprobe attachment
type UprobeTarget struct {
	Binary      string
	Symbol      string
	Type        string // "database", "cache", "gc", "custom"
	Description string
}

// AppSpecificTracker manages application-specific eBPF programs
type AppSpecificTracker struct {
	objects       *app_specificObjects
	links         []link.Link
	ringbufReader *ringbuf.Reader
	containerMap  *ectx.ContainerMapper

	// Active uprobe targets
	uprobeTargets []UprobeTarget

	// Event handlers
	onApplicationMetrics func(*pb.ApplicationMetrics)
}

// NewAppSpecificTracker creates a new application-specific tracker
func NewAppSpecificTracker(containerMap *ectx.ContainerMapper) (*AppSpecificTracker, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock: %w", err)
	}

	objects := &app_specificObjects{}
	if err := loadApp_specificObjects(objects, nil); err != nil {
		return nil, fmt.Errorf("failed to load app-specific eBPF objects: %w", err)
	}

	tracker := &AppSpecificTracker{
		objects:      objects,
		containerMap: containerMap,
		uprobeTargets: []UprobeTarget{
			// Common database symbols
			{Binary: "/usr/bin/postgres", Symbol: "exec_simple_query", Type: "database", Description: "PostgreSQL query execution"},
			{Binary: "/usr/bin/mysqld", Symbol: "mysql_execute_command", Type: "database", Description: "MySQL query execution"},

			// Redis cache operations
			{Binary: "/usr/bin/redis-server", Symbol: "lookupCommand", Type: "cache", Description: "Redis command lookup"},
			{Binary: "/usr/bin/memcached", Symbol: "process_get_command", Type: "cache", Description: "Memcached GET operation"},

			// Go runtime symbols (if running Go applications)
			{Binary: "/usr/local/go/bin/go", Symbol: "runtime.GC", Type: "gc", Description: "Go garbage collection"},
			{Binary: "", Symbol: "runtime.GC", Type: "gc", Description: "Go GC (dynamic)"},
		},
	}

	return tracker, nil
}

// Start begins application-specific tracking
func (ast *AppSpecificTracker) Start(ctx context.Context) error {
	// Set up ring buffer reader
	if err := ast.setupRingBuffer(ctx); err != nil {
		return fmt.Errorf("failed to setup ring buffer: %w", err)
	}

	// Attach dynamic uprobes for detected applications
	if err := ast.attachDynamicUprobes(); err != nil {
		log.Warn().Err(err).Msg("Failed to attach some dynamic uprobes, continuing with available attachments")
	}

	log.Info().Msg("Application-specific tracker started")
	return nil
}

// attachDynamicUprobes attempts to attach uprobes to detected application binaries
func (ast *AppSpecificTracker) attachDynamicUprobes() error {
	// This would ideally scan running processes and detect which applications are running
	// For now, we'll try to attach to common symbols

	for _, target := range ast.uprobeTargets {
		if target.Binary == "" {
			continue // Skip dynamic targets for now
		}

		if err := ast.attachUprobeTarget(target); err != nil {
			log.Debug().
				Err(err).
				Str("binary", target.Binary).
				Str("symbol", target.Symbol).
				Msg("Failed to attach uprobe target")
			continue
		}

		log.Info().
			Str("binary", target.Binary).
			Str("symbol", target.Symbol).
			Str("type", target.Type).
			Msg("Successfully attached uprobe")
	}

	return nil
}

// attachUprobeTarget attaches uprobes for a specific target
func (ast *AppSpecificTracker) attachUprobeTarget(target UprobeTarget) error {
	// Open the executable
	ex, err := link.OpenExecutable(target.Binary)
	if err != nil {
		return fmt.Errorf("failed to open executable %s: %w", target.Binary, err)
	}

	// Determine which eBPF programs to attach based on type
	switch target.Type {
	case "database":
		// Attach database query start/end probes
		startLink, err := ex.Uprobe(target.Symbol, ast.objects.TraceDbQueryStart, nil)
		if err != nil {
			return fmt.Errorf("failed to attach database start uprobe: %w", err)
		}
		ast.links = append(ast.links, startLink)

		endLink, err := ex.Uretprobe(target.Symbol, ast.objects.TraceDbQueryEnd, nil)
		if err != nil {
			return fmt.Errorf("failed to attach database end uprobe: %w", err)
		}
		ast.links = append(ast.links, endLink)

	case "cache":
		// Attach cache operation probes
		startLink, err := ex.Uprobe(target.Symbol, ast.objects.TraceCacheGetStart, nil)
		if err != nil {
			return fmt.Errorf("failed to attach cache start uprobe: %w", err)
		}
		ast.links = append(ast.links, startLink)

		endLink, err := ex.Uretprobe(target.Symbol, ast.objects.TraceCacheGetEnd, nil)
		if err != nil {
			return fmt.Errorf("failed to attach cache end uprobe: %w", err)
		}
		ast.links = append(ast.links, endLink)

	case "gc":
		// Attach GC probes
		startLink, err := ex.Uprobe(target.Symbol, ast.objects.TraceGoGcStart, nil)
		if err != nil {
			return fmt.Errorf("failed to attach GC start uprobe: %w", err)
		}
		ast.links = append(ast.links, startLink)

		endLink, err := ex.Uretprobe(target.Symbol, ast.objects.TraceGoGcEnd, nil)
		if err != nil {
			return fmt.Errorf("failed to attach GC end uprobe: %w", err)
		}
		ast.links = append(ast.links, endLink)

	case "custom":
		// Attach custom metric probe
		customLink, err := ex.Uprobe(target.Symbol, ast.objects.TraceCustomMetric, nil)
		if err != nil {
			return fmt.Errorf("failed to attach custom metric uprobe: %w", err)
		}
		ast.links = append(ast.links, customLink)
	}

	return nil
}

// setupRingBuffer sets up the ring buffer for real-time events
func (ast *AppSpecificTracker) setupRingBuffer(ctx context.Context) error {
	reader, err := ringbuf.NewReader(ast.objects.AppEvents)
	if err != nil {
		return fmt.Errorf("failed to create ring buffer reader: %w", err)
	}
	ast.ringbufReader = reader

	// Start ring buffer processing
	go ast.processRingBufferEvents(ctx)

	return nil
}

// processRingBufferEvents processes real-time application events
func (ast *AppSpecificTracker) processRingBufferEvents(ctx context.Context) {
	defer ast.ringbufReader.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			record, err := ast.ringbufReader.Read()
			if err != nil {
				if err != ringbuf.ErrClosed {
					log.Error().Err(err).Msg("Failed to read from app events ring buffer")
				}
				continue
			}

			if len(record.RawSample) < int(unsafe.Sizeof(AppEvent{})) {
				log.Warn().Int("size", len(record.RawSample)).Msg("Invalid app event size")
				continue
			}

			// Parse the event
			event := (*AppEvent)(unsafe.Pointer(&record.RawSample[0]))

			// Process the event
			ast.processAppEvent(event)
		}
	}
}

// processAppEvent processes a single application event
func (ast *AppSpecificTracker) processAppEvent(event *AppEvent) {
	containerInfo, exists := ast.containerMap.GetContainerByCgroupID(event.CgroupID)
	if !exists {
		return
	}

	// Convert details to string
	details := string(event.Details[:])
	details = strings.TrimRight(details, "\x00")

	// Log significant events
	if event.DurationUS > 1000000 { // Events taking longer than 1 second
		eventType := "unknown"
		switch event.EventType {
		case 0:
			eventType = "database_query"
		case 1:
			eventType = "cache_operation"
		case 2:
			eventType = "gc_event"
		case 3:
			eventType = "custom_metric"
		}

		log.Debug().
			Str("container", containerInfo.ContainerID).
			Str("app", containerInfo.AppName).
			Str("event_type", eventType).
			Uint64("duration_us", event.DurationUS).
			Uint64("value", event.Value).
			Str("details", details).
			Msg("Significant application event")
	}
}

// Poll implements the CollectorProgram interface
func (ast *AppSpecificTracker) Poll(ctx context.Context) error {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := ast.collectAndEmitMetrics(); err != nil {
				log.Error().Err(err).Msg("Failed to collect application metrics")
			}
		}
	}
}

// collectAndEmitMetrics reads current metrics from eBPF maps and emits them
func (ast *AppSpecificTracker) collectAndEmitMetrics() error {
	// Collect metrics from all containers
	containerMetrics := make(map[uint64]*pb.ApplicationMetrics)

	// Collect database metrics
	if err := ast.collectDatabaseMetrics(containerMetrics); err != nil {
		log.Error().Err(err).Msg("Failed to collect database metrics")
	}

	// Collect cache metrics
	if err := ast.collectCacheMetrics(containerMetrics); err != nil {
		log.Error().Err(err).Msg("Failed to collect cache metrics")
	}

	// Collect Go metrics
	if err := ast.collectGoMetrics(containerMetrics); err != nil {
		log.Error().Err(err).Msg("Failed to collect Go metrics")
	}

	// Emit metrics for each container
	for _, metrics := range containerMetrics {
		if ast.onApplicationMetrics != nil {
			ast.onApplicationMetrics(metrics)
		}
	}

	return nil
}

// collectDatabaseMetrics collects database metrics from eBPF maps
func (ast *AppSpecificTracker) collectDatabaseMetrics(containerMetrics map[uint64]*pb.ApplicationMetrics) error {
	var nextKey uint64
	var dbMetrics DatabaseMetrics

	iter := ast.objects.DbMetricsMap.Iterate()
	for iter.Next(&nextKey, &dbMetrics) {
		cgroupID := nextKey

		if dbMetrics.QueryCount == 0 {
			continue
		}

		// Get or create application metrics for this container
		appMetrics := containerMetrics[cgroupID]
		if appMetrics == nil {
			now := time.Now()
			windowStart := now.Add(-15 * time.Second)
			appMetrics = &pb.ApplicationMetrics{
				WindowStart: timestampFromTime(windowStart),
				WindowEnd:   timestampFromTime(now),
			}
			containerMetrics[cgroupID] = appMetrics
		}

		// Calculate average query time
		var avgQueryTime float64
		if dbMetrics.QueryCount > 0 {
			avgQueryTime = float64(dbMetrics.QueryTimeTotal) / float64(dbMetrics.QueryCount)
		}

		// Calculate average transaction time
		var avgTransactionTime float64
		if dbMetrics.Transactions > 0 {
			avgTransactionTime = float64(dbMetrics.TransactionTimeTotal) / float64(dbMetrics.Transactions)
		}

		// Create database metrics
		appMetrics.Database = &pb.DatabaseMetrics{
			QueryCount:           dbMetrics.QueryCount,
			AvgQueryTimeUs:       avgQueryTime,
			SlowQueries:          dbMetrics.SlowQueries,
			SlowQueryThresholdUs: 1000000, // 1 second
			QueryTypes: map[string]uint64{
				"SELECT": dbMetrics.QueryCount * 60 / 100, // Estimate
				"INSERT": dbMetrics.QueryCount * 20 / 100,
				"UPDATE": dbMetrics.QueryCount * 15 / 100,
				"DELETE": dbMetrics.QueryCount * 5 / 100,
			},
			ActiveConnections:     dbMetrics.ActiveConnections,
			ConnectionTimeouts:    dbMetrics.ConnectionTimeouts,
			Transactions:          dbMetrics.Transactions,
			Rollbacks:             dbMetrics.Rollbacks,
			AvgTransactionTimeUs:  avgTransactionTime,
			SlowQuerySamples:      []*pb.SlowQuery{}, // TODO: Implement slow query sampling
		}
	}

	return iter.Err()
}

// collectCacheMetrics collects cache metrics from eBPF maps
func (ast *AppSpecificTracker) collectCacheMetrics(containerMetrics map[uint64]*pb.ApplicationMetrics) error {
	var nextKey uint64
	var cacheMetrics CacheMetrics

	iter := ast.objects.CacheMetricsMap.Iterate()
	for iter.Next(&nextKey, &cacheMetrics) {
		cgroupID := nextKey

		if cacheMetrics.CacheGets == 0 && cacheMetrics.CacheSets == 0 {
			continue
		}

		// Get or create application metrics for this container
		appMetrics := containerMetrics[cgroupID]
		if appMetrics == nil {
			now := time.Now()
			windowStart := now.Add(-15 * time.Second)
			appMetrics = &pb.ApplicationMetrics{
				WindowStart: timestampFromTime(windowStart),
				WindowEnd:   timestampFromTime(now),
			}
			containerMetrics[cgroupID] = appMetrics
		}

		// Calculate hit rate
		totalRequests := cacheMetrics.CacheHits + cacheMetrics.CacheMisses
		var hitRate float64
		if totalRequests > 0 {
			hitRate = float64(cacheMetrics.CacheHits) / float64(totalRequests) * 100
		}

		// Calculate average latencies
		var avgGetLatency, avgSetLatency float64
		if cacheMetrics.CacheGets > 0 {
			avgGetLatency = float64(cacheMetrics.GetLatencyTotal) / float64(cacheMetrics.CacheGets)
		}
		if cacheMetrics.CacheSets > 0 {
			avgSetLatency = float64(cacheMetrics.SetLatencyTotal) / float64(cacheMetrics.CacheSets)
		}

		// Calculate cache utilization
		var cacheUtilization float64
		if cacheMetrics.CacheSizeBytes > 0 {
			// This would need actual capacity information
			estimatedCapacity := cacheMetrics.CacheSizeBytes * 2 // Rough estimate
			cacheUtilization = float64(cacheMetrics.CacheSizeBytes) / float64(estimatedCapacity) * 100
		}

		// Create cache metrics
		appMetrics.Cache = &pb.CacheMetrics{
			CacheHits:         cacheMetrics.CacheHits,
			CacheMisses:       cacheMetrics.CacheMisses,
			HitRate:           hitRate,
			CacheSets:         cacheMetrics.CacheSets,
			CacheGets:         cacheMetrics.CacheGets,
			CacheDeletes:      cacheMetrics.CacheDeletes,
			CacheEvictions:    cacheMetrics.CacheEvictions,
			AvgGetLatencyUs:   avgGetLatency,
			AvgSetLatencyUs:   avgSetLatency,
			CacheSizeBytes:    cacheMetrics.CacheSizeBytes,
			CacheUtilization:  cacheUtilization,
			CacheType:         "redis", // This could be detected dynamically
		}
	}

	return iter.Err()
}

// collectGoMetrics collects Go runtime metrics from eBPF maps
func (ast *AppSpecificTracker) collectGoMetrics(containerMetrics map[uint64]*pb.ApplicationMetrics) error {
	var nextKey uint64
	var goMetrics GoMetrics

	iter := ast.objects.GoMetricsMap.Iterate()
	for iter.Next(&nextKey, &goMetrics) {
		cgroupID := nextKey

		if goMetrics.GCCycles == 0 {
			continue
		}

		// Get or create application metrics for this container
		appMetrics := containerMetrics[cgroupID]
		if appMetrics == nil {
			now := time.Now()
			windowStart := now.Add(-15 * time.Second)
			appMetrics = &pb.ApplicationMetrics{
				WindowStart: timestampFromTime(windowStart),
				WindowEnd:   timestampFromTime(now),
			}
			containerMetrics[cgroupID] = appMetrics
		}

		// Calculate average GC pause time
		var avgGCPause float64
		if goMetrics.GCCycles > 0 {
			avgGCPause = float64(goMetrics.GCPauseTimeUS) / float64(goMetrics.GCCycles)
		}

		// Create language metrics structure if needed
		if appMetrics.Language == nil {
			appMetrics.Language = &pb.LanguageMetrics{}
		}

		// Create Go metrics
		appMetrics.Language.GoMetrics = &pb.GoMetrics{
			Goroutines:         goMetrics.Goroutines,
			GoroutineStackSize: goMetrics.GoroutineStackSize,
			GcCycles:           goMetrics.GCCycles,
			GcPauseTimeUs:      avgGCPause,
			HeapSize:           goMetrics.HeapSize,
			HeapAlloc:          goMetrics.HeapAlloc,
			HeapIdle:           goMetrics.HeapIdle,
			ChannelSends:       goMetrics.ChannelSends,
			ChannelReceives:    goMetrics.ChannelReceives,
			BlockedChannels:    goMetrics.BlockedChannels,
		}
	}

	return iter.Err()
}

// SetApplicationMetricsHandler sets the callback for application metrics
func (ast *AppSpecificTracker) SetApplicationMetricsHandler(handler func(*pb.ApplicationMetrics)) {
	ast.onApplicationMetrics = handler
}

// AddUprobeTarget adds a new uprobe target for dynamic attachment
func (ast *AppSpecificTracker) AddUprobeTarget(target UprobeTarget) error {
	ast.uprobeTargets = append(ast.uprobeTargets, target)

	// Try to attach immediately if we're already running
	if ast.objects != nil {
		return ast.attachUprobeTarget(target)
	}

	return nil
}

// Close releases all resources
func (ast *AppSpecificTracker) Close() error {
	// Close ring buffer reader
	if ast.ringbufReader != nil {
		ast.ringbufReader.Close()
	}

	// Detach all links
	for _, l := range ast.links {
		if err := l.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close eBPF link")
		}
	}

	// Close eBPF objects
	if ast.objects != nil {
		ast.objects.Close()
	}

	log.Info().Msg("Application-specific tracker closed")
	return nil
}

// Helper functions

func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}