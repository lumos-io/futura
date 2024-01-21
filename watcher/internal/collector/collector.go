package collector

// aggregate data from different sources
// 1. k8s
// 2. containerd (TODO)
// 3. ebpf
// 4. cgroup (TODO)
// 5. docker (TODO)

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"

	"github.com/opisvigilant/futura/watcher/internal/ebpf"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/l7_req"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/proc"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/tcp_state"
	"github.com/opisvigilant/futura/watcher/internal/handlers"

	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/internal/models"

	"github.com/patrickmn/go-cache"
	"k8s.io/apimachinery/pkg/types"
)

type Collector struct {
	ctx context.Context

	ec *ebpf.EbpfCollector

	// store the service map
	clusterInfo *ClusterInfo

	// send data to datastore
	eventsHandler handlers.Handler

	// http2 ch
	h2ChMu sync.RWMutex
	h2Ch   map[uint32]chan *l7_req.L7Event // pid -> ch

	h2ParserMu sync.RWMutex
	h2Parsers  map[string]*http2Parser // pid-fd -> http2Parser

	liveProcessesMu sync.RWMutex
	liveProcesses   map[uint32]struct{} // pid -> struct{}
}

// We need to keep track of the following
// in order to build find relationships between
// connections and pods/services

type SockInfo struct {
	Pid   uint32 `json:"pid"`
	Fd    uint64 `json:"fd"`
	Saddr string `json:"saddr"`
	Sport uint16 `json:"sport"`
	Daddr string `json:"daddr"`
	Dport uint16 `json:"dport"`
}

type http2Parser struct {
	// // Framer is the HTTP/2 framer to use.
	// framer *http2.Framer
	// // framer.ReadFrame() returns a frame, which is a struct

	// http2 request and response dynamic tables are separate
	// 2 decoders are needed
	// https://httpwg.org/specs/rfc7541.html#encoding.context
	clientHpackDecoder *hpack.Decoder
	serverHpackDecoder *hpack.Decoder
}

// type SocketMap
type SocketMap struct {
	mu sync.RWMutex
	M  map[uint64]*SocketLine `json:"fdToSockLine"` // fd -> SockLine
}

type ClusterInfo struct {
	mu                    sync.RWMutex
	PodIPToPodUid         map[string]types.UID `json:"podIPToPodUid"`
	ServiceIPToServiceUid map[string]types.UID `json:"serviceIPToServiceUid"`

	// Pid -> SocketMap
	// pid -> fd -> {saddr, sport, daddr, dport}
	PidToSocketMap map[uint32]*SocketMap `json:"pidToSocketMap"`
}

// If we have information from the container runtimes
// we would have pid's of the containers within the pod
// and we can use that to find the podUid directly

// If we don't have the pid's of the containers
// we can use the following to find the podUid
// {saddr+sport} -> search in podIPToPodUid -> podUid
// {daddr+dport} -> search in serviceIPToServiceUid -> serviceUid
// or
// {daddr+dport} -> search in podIPToPodUid -> podUid

var (
	// default exponential backoff (*2)
	// when retryLimit is increased, we are blocking the events that we wait it to be processed more
	retryInterval = 400 * time.Millisecond
	retryLimit    = 5
	// 400 + 800 + 1600 + 3200 + 6400 = 12400 ms

	defaultExpiration = 5 * time.Minute
	purgeTime         = 10 * time.Minute
)

var reverseDnsCache *cache.Cache

var re *regexp.Regexp

func init() {
	reverseDnsCache = cache.New(defaultExpiration, purgeTime)

	keywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "FROM", "WHERE", "JOIN", "INNER", "OUTER", "LEFT", "RIGHT", "GROUP", "BY", "ORDER", "HAVING", "UNION", "ALL", "BEGIN", "COMMIT"}

	// Case-insensitive matching
	re = regexp.MustCompile(strings.Join(keywords, "|"))
}

// Check if a string contains SQL keywords
func containsSQLKeywords(input string) bool {
	return re.MatchString(strings.ToUpper(input))
}

func NewCollector(parentCtx context.Context, eventHandler handlers.Handler, ec *ebpf.EbpfCollector) *Collector {
	ctx, _ := context.WithCancel(parentCtx)
	clusterInfo := &ClusterInfo{
		PodIPToPodUid:         map[string]types.UID{},
		ServiceIPToServiceUid: map[string]types.UID{},
		PidToSocketMap:        make(map[uint32]*SocketMap, 0),
	}

	a := &Collector{
		ctx:           ctx,
		ec:            ec,
		eventsHandler: eventHandler,
		clusterInfo:   clusterInfo,
		h2Ch:          make(map[uint32]chan *l7_req.L7Event),
		h2Parsers:     make(map[string]*http2Parser),
		liveProcesses: make(map[uint32]struct{}),
	}

	go a.clearSocketLines(ctx)

	return a
}

// ec.EbpfEvents()
func (c *Collector) Run(k8sChan <-chan interface{}, ebpfChan <-chan interface{}) {
	go func() {
		// get all alive processes, populate liveProcesses
		cmd := exec.Command("ps", "-e", "-o", "pid=")
		output, err := cmd.Output()
		if err != nil {
			logger.Logger().Fatal().Err(err).Msg("error getting all alive processes")
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					pid := fields[0]
					pidInt, err := strconv.Atoi(pid)
					if err != nil {
						logger.Logger().Error().Err(err).Msgf("error converting pid to int %s", pid)
						continue
					}
					c.liveProcesses[uint32(pidInt)] = struct{}{}
				}
			}
		}
	}()
	go func() {
		// every 5 minutes, check alive processes, and clear the ones left behind
		// since we process events concurrently, some short-lived processes exit event can come before exec events
		// this causes zombie http2 workers

		t := time.NewTicker(2 * time.Minute)
		defer t.Stop()

		for range t.C {
			c.liveProcessesMu.Lock()

			for pid, _ := range c.liveProcesses {
				// https://man7.org/linux/man-pages/man2/kill.2.html
				//    If sig is 0, then no signal is sent, but existence and permission
				//    checks are still performed; this can be used to check for the
				//    existence of a process ID or process group ID that the caller is
				//    permitted to signal.

				err := syscall.Kill(int(pid), 0)
				if err != nil {
					// pid does not exist
					delete(c.liveProcesses, pid)

					c.clusterInfo.mu.Lock()
					delete(c.clusterInfo.PidToSocketMap, pid)
					c.clusterInfo.mu.Unlock()

					// close http2Worker if exist
					c.stopHttp2Worker(pid)
				}
			}
			c.liveProcessesMu.Unlock()
		}
	}()

	go c.processk8s(k8sChan)

	numWorker := 100
	for i := 0; i < numWorker; i++ {
		go c.processEbpf(c.ctx, ebpfChan)
	}
}

func (c *Collector) processk8s(k8sChan <-chan interface{}) {
	c.eventsHandler.HandleKubernetesEvent(k8sChan)
}

func (c *Collector) processEbpf(ctx context.Context, ebpfChan <-chan interface{}) {
	stop := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(stop)
	}()

	for data := range ebpfChan {
		select {
		case <-stop:
			logger.Logger().Info().Msg("processEbpf exiting...")
			return
		default:
			bpfEvent, ok := data.(ebpf.BpfEvent)
			if !ok {
				logger.Logger().Error().Interface("ebpfData", data).Msg("error casting ebpf event")
				continue
			}
			switch bpfEvent.Type() {
			case tcp_state.TCP_CONNECT_EVENT:
				d := data.(*tcp_state.TcpConnectEvent) // copy data's value
				go c.processTcpConnect(d)
			case l7_req.L7_EVENT:
				d := data.(*l7_req.L7Event) // copy data's value
				go c.processL7(ctx, d)
			case proc.PROC_EVENT:
				d := data.(*proc.ProcEvent) // copy data's value
				if d.Type_ == proc.EVENT_PROC_EXEC {
					go c.processExec(d)
				} else if d.Type_ == proc.EVENT_PROC_EXIT {
					go c.processExit(d.Pid)
				}
			case l7_req.TRACE_EVENT:
				d := data.(*l7_req.TraceEvent)
				c.eventsHandler.PersistTraceEvent(d)
			}
		}
	}
}

func (a *Collector) processExec(d *proc.ProcEvent) {
	a.liveProcessesMu.Lock()
	a.liveProcesses[d.Pid] = struct{}{}
	a.liveProcessesMu.Unlock()
}

func (a *Collector) processExit(pid uint32) {
	a.liveProcessesMu.Lock()
	delete(a.liveProcesses, pid)
	a.liveProcessesMu.Unlock()

	a.clusterInfo.mu.Lock()
	delete(a.clusterInfo.PidToSocketMap, pid)
	a.clusterInfo.mu.Unlock()

	// close http2Worker if exist
	a.stopHttp2Worker(pid)
}

func (a *Collector) stopHttp2Worker(pid uint32) {
	a.h2ChMu.Lock()
	defer a.h2ChMu.Unlock()
	if ch, ok := a.h2Ch[pid]; ok {
		close(ch)
		delete(a.h2Ch, pid)

		a.h2ParserMu.Lock()
		for key, parser := range a.h2Parsers {
			// h2Parsers  map[string]*http2Parser // pid-fd -> http2Parser
			if strings.HasPrefix(key, fmt.Sprint(pid)) {
				parser.clientHpackDecoder.Close()
				parser.serverHpackDecoder.Close()

				delete(a.h2Parsers, key)
			}
		}
		a.h2ParserMu.Unlock()
	}
}

func (a *Collector) processTcpConnect(d *tcp_state.TcpConnectEvent) {
	go a.ec.ListenForEncryptedReqs(d.Pid)
	if d.Type_ == tcp_state.EVENT_TCP_ESTABLISHED {
		// filter out localhost connections
		if d.SAddr == "127.0.0.1" || d.DAddr == "127.0.0.1" {
			return
		}

		var sockMap *SocketMap
		var ok bool

		a.clusterInfo.mu.RLock() // lock for reading
		sockMap, ok = a.clusterInfo.PidToSocketMap[d.Pid]
		a.clusterInfo.mu.RUnlock() // unlock for reading
		if !ok {
			sockMap = &SocketMap{
				M:  make(map[uint64]*SocketLine),
				mu: sync.RWMutex{},
			}
			a.clusterInfo.mu.Lock() // lock for writing
			a.clusterInfo.PidToSocketMap[d.Pid] = sockMap
			a.clusterInfo.mu.Unlock() // unlock for writing
		}

		var skLine *SocketLine

		sockMap.mu.RLock() // lock for reading
		skLine, ok = sockMap.M[d.Fd]
		sockMap.mu.RUnlock() // unlock for reading

		if !ok {
			skLine = NewSocketLine(d.Pid, d.Fd)
			sockMap.mu.Lock() // lock for writing
			sockMap.M[d.Fd] = skLine
			sockMap.mu.Unlock() // unlock for writing
		}

		skLine.AddValue(
			d.Timestamp, // get connection timestamp from ebpf
			&SockInfo{
				Pid:   d.Pid,
				Fd:    d.Fd,
				Saddr: d.SAddr,
				Sport: d.SPort,
				Daddr: d.DAddr,
				Dport: d.DPort,
			},
		)
	} else if d.Type_ == tcp_state.EVENT_TCP_CLOSED {
		var sockMap *SocketMap
		var ok bool

		a.clusterInfo.mu.RLock() // lock for reading
		sockMap, ok = a.clusterInfo.PidToSocketMap[d.Pid]
		a.clusterInfo.mu.RUnlock() // unlock for reading

		if !ok {
			sockMap = &SocketMap{
				M:  make(map[uint64]*SocketLine),
				mu: sync.RWMutex{},
			}

			a.clusterInfo.mu.Lock() // lock for writing
			a.clusterInfo.PidToSocketMap[d.Pid] = sockMap
			a.clusterInfo.mu.Unlock() // unlock for writing
			return
		}

		var skLine *SocketLine

		sockMap.mu.RLock() // lock for reading
		skLine, ok = sockMap.M[d.Fd]
		sockMap.mu.RUnlock() // unlock for reading

		if !ok {
			return
		}

		// If connection is established before, add the close event
		skLine.AddValue(
			d.Timestamp, // get connection close timestamp from ebpf
			nil,         // closed
		)

		// remove h2Parser if exists
		a.h2ParserMu.Lock()
		key := a.getConnKey(d.Pid, d.Fd)
		h2Parser, ok := a.h2Parsers[key]
		if ok {
			h2Parser.clientHpackDecoder.Close()
			h2Parser.serverHpackDecoder.Close()
		}
		delete(a.h2Parsers, key)
		a.h2ParserMu.Unlock()

	}
}

func parseHttpPayload(request string) (method string, path string, httpVersion string, hostHeader string) {
	// Find the first space character
	lines := strings.Split(request, "\n")
	parts := strings.Split(lines[0], " ")
	if len(parts) >= 3 {
		method = parts[0]
		path = parts[1]
		httpVersion = parts[2]
	}

	for _, line := range lines[1:] {
		// find Host header
		if strings.HasPrefix(line, "Host:") {
			hostParts := strings.Split(line, " ")
			if len(hostParts) >= 2 {
				hostHeader = hostParts[1]
				hostHeader = strings.TrimSuffix(hostHeader, "\r")
				break
			}
		}
	}

	return method, path, httpVersion, hostHeader
}

type FrameArrival struct {
	ClientHeadersFrameArrived bool
	ServerHeadersFrameArrived bool
	ServerDataFrameArrived    bool
	event                     *l7_req.L7Event // l7 event that carries server data frame
	req                       *models.Request

	statusCode uint32
	grpcStatus uint32
}

// called once per pid
func (a *Collector) processHttp2Frames(pid uint32, ch chan *l7_req.L7Event) {
	mu := sync.RWMutex{}

	createFrameKey := func(fd uint64, streamId uint32) string {
		return fmt.Sprintf("%d-%d", fd, streamId)
	}
	// fd-streamId -> frame
	frames := make(map[string]*FrameArrival)

	done := make(chan bool, 1)

	go func() {
		t := time.NewTicker(1 * time.Minute)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				mu.Lock()
				for key, f := range frames {
					if f.ClientHeadersFrameArrived && !f.ServerHeadersFrameArrived {
						delete(frames, key)
					} else if !f.ClientHeadersFrameArrived && f.ServerHeadersFrameArrived {
						delete(frames, key)
					}
				}
				mu.Unlock()
			case <-done:
				return
			}
		}
	}()

	persistReq := func(d *l7_req.L7Event, req *models.Request, statusCode uint32, grpcStatus uint32) {
		if req.Method == "" || req.Path == "" {
			// if we couldn't parse the request, discard
			// this is possible because of hpack dynamic table, we can't parse the request until a new connection is established

			// TODO: check if duplicate processing happens for the same request at some point on processing
			// magic message can be used to identify the connection on ebpf side
			// when adjustment is made on ebpf side, we can remove this check
			return
		}

		skInfo := a.findRelatedSocket(a.ctx, d)
		if skInfo == nil {
			return
		}

		req.StartTime = d.EventReadTime
		req.Latency = d.WriteTimeNs - req.Latency
		req.Completed = true
		req.FromIP = skInfo.Saddr
		req.ToIP = skInfo.Daddr
		req.Tls = d.Tls
		req.FromPort = skInfo.Sport
		req.ToPort = skInfo.Dport
		req.FailReason = ""
		if req.Protocol == "" {
			req.Protocol = "HTTP2"
			if req.Tls {
				req.Protocol = "HTTPS"
			}
			req.StatusCode = statusCode
		} else if req.Protocol == "gRPC" {
			req.StatusCode = grpcStatus
		}

		// toUID is set to :authority header in client frame
		err := a.setFromTo(skInfo, d, req, req.ToUID)
		if err != nil {
			return
		}

		a.eventsHandler.PersistRequest(req)
	}

	parseFrameHeader := func(buf []byte) http2.FrameHeader {
		// http2/frame.go/readFrameHeader
		// to avoid copy op, we read the frame header manually here
		return http2.FrameHeader{
			Length:   (uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])),
			Type:     http2.FrameType(buf[3]),
			Flags:    http2.Flags(buf[4]),
			StreamID: binary.BigEndian.Uint32(buf[5:]) & (1<<31 - 1),
		}
	}

	for d := range ch {
		// Normally we tried to use http2.Framer to parse frames but
		// http2.Framer spends too much memory and cpu reading frames
		// golang.org/x/net/http2.(*Framer).ReadFrame /go/pkg/mod/golang.org/x/net@v0.12.0/http2/frame.go:505
		// golang.org/x/net/http2.NewFramer.func2 /go/pkg/mod/golang.org/x/net@v0.12.0/http2/frame.go:444
		// getReadBuf is called for every ReadFrame call and allocates a new buffer
		// Additionally, later on io.ReadFull is called to copy the frame to the buffer
		// both cpu and memory intensive ops

		// framer := http2.NewFramer(nil, bytes.NewReader(d.Payload[0:d.PayloadSize]))
		// framer.SetReuseFrames()

		buf := d.Payload[:d.PayloadSize]
		fd := d.Fd
		var offset uint32 = 0

		a.h2ParserMu.RLock()
		h2Parser := a.h2Parsers[a.getConnKey(d.Pid, d.Fd)]
		a.h2ParserMu.RUnlock()
		if h2Parser == nil {
			a.h2ParserMu.Lock()
			h2Parser = &http2Parser{
				clientHpackDecoder: hpack.NewDecoder(4096, nil),
				serverHpackDecoder: hpack.NewDecoder(4096, nil),
			}
			a.h2Parsers[a.getConnKey(d.Pid, d.Fd)] = h2Parser
			a.h2ParserMu.Unlock()
		}

		// parse frame
		// https://httpwg.org/specs/rfc7540.html#rfc.section.4.1
		if d.Method == l7_req.CLIENT_FRAME {
			for {
				// can be multiple frames in the payload

				// http2/frame.go/readFrameHeader
				// to avoid copy op, we read the frame header manually here

				if len(buf)-int(offset) < 9 {
					break
				}

				fh := parseFrameHeader(buf[offset:])

				// frame header consists of 9 bytes
				offset += 9

				endOfFrame := offset + fh.Length
				// since we read constant 1024 bytes from the kernel
				// we need to check left over bytes are enough to read
				if len(buf) < int(endOfFrame) {
					break
				}

				// skip if not headers frame
				if fh.Type != http2.FrameHeaders {
					offset = endOfFrame
					continue
				}

				streamId := fh.StreamID
				key := createFrameKey(fd, streamId)
				mu.Lock()
				if _, ok := frames[key]; !ok {
					frames[key] = &FrameArrival{
						ClientHeadersFrameArrived: true,
						req:                       &models.Request{},
					}
				}

				fa := frames[key]
				fa.ClientHeadersFrameArrived = true
				fa.req.Latency = d.WriteTimeNs // set latency to write time here, will be updated later

				// Process client headers frame
				reqHeaderSet := func(req *models.Request) func(hf hpack.HeaderField) {
					return func(hf hpack.HeaderField) {
						switch hf.Name {
						case ":method":
							if req.Method == "" {
								req.Method = hf.Value
							}
						case ":path":
							if req.Path == "" {
								req.Path = hf.Value
							}
						case ":authority":
							if req.ToUID == "" {
								req.ToUID = hf.Value
							}
						case "content-type":
							if req.Protocol == "" {
								if strings.HasPrefix(hf.Value, "application/grpc") {
									req.Protocol = "gRPC"
								}
							}
						}
					}
				}
				h2Parser.clientHpackDecoder.SetEmitFunc(reqHeaderSet(fa.req))

				// if ReadFrame were used, f.HeaderBlockFragment()
				h2Parser.clientHpackDecoder.Write(buf[offset:endOfFrame])

				offset = endOfFrame

				if fa.ServerHeadersFrameArrived {
					req := *fa.req
					go persistReq(d, &req, fa.statusCode, fa.grpcStatus)
					delete(frames, key)
				}
				mu.Unlock()
				break
			}
		} else if d.Method == l7_req.SERVER_FRAME {
			for {
				if len(buf)-int(offset) < 9 {
					break
				}
				// can be multiple frames in the payload
				fh := parseFrameHeader(buf[offset:])
				offset += 9

				endOfFrame := offset + fh.Length
				// since we read constant 1024 bytes from the kernel
				// we need to check left over bytes are enough to read
				if len(buf) < int(endOfFrame) {
					break
				}

				streamId := fh.StreamID
				key := createFrameKey(fd, streamId)

				if fh.Type != http2.FrameHeaders {
					offset = endOfFrame
					continue
				}

				if fh.Type == http2.FrameHeaders {
					mu.Lock()
					if _, ok := frames[key]; !ok {
						frames[key] = &FrameArrival{
							ServerHeadersFrameArrived: true,
							req:                       &models.Request{},
						}
					}
					fa := frames[key]
					fa.ServerHeadersFrameArrived = true
					// Process server headers frame
					respHeaderSet := func(req *models.Request) func(hf hpack.HeaderField) {
						return func(hf hpack.HeaderField) {
							switch hf.Name {
							case ":status":
								s, _ := strconv.Atoi(hf.Value)
								fa.statusCode = uint32(s)
							case "grpc-status":
								s, _ := strconv.Atoi(hf.Value)
								fa.grpcStatus = uint32(s)
							}
						}
					}
					h2Parser.serverHpackDecoder.SetEmitFunc(respHeaderSet(fa.req))
					h2Parser.serverHpackDecoder.Write(buf[offset:endOfFrame])

					if fa.ClientHeadersFrameArrived {
						req := *fa.req
						go persistReq(d, &req, fa.statusCode, fa.grpcStatus)
						delete(frames, key)
					}
					mu.Unlock()
					break
				}
			}
		} else {
			logger.Logger().Error().Msg("unknown http2 frame type")
			continue
		}
	}

	done <- true // signal cleaning goroutine
}

func (a *Collector) setFromTo(skInfo *SockInfo, d *l7_req.L7Event, reqDto *models.Request, hostHeader string) error {
	// find pod info
	a.clusterInfo.mu.RLock() // lock for reading
	podUid, ok := a.clusterInfo.PodIPToPodUid[skInfo.Saddr]
	a.clusterInfo.mu.RUnlock() // unlock for reading
	if !ok {
		return fmt.Errorf("error finding pod with sockets saddr")
	}

	reqDto.FromUID = string(podUid)
	reqDto.FromType = "pod"
	reqDto.FromPort = skInfo.Sport
	reqDto.ToPort = skInfo.Dport

	// find service info
	a.clusterInfo.mu.RLock() // lock for reading
	svcUid, ok := a.clusterInfo.ServiceIPToServiceUid[skInfo.Daddr]
	a.clusterInfo.mu.RUnlock() // unlock for reading

	if ok {
		reqDto.ToUID = string(svcUid)
		reqDto.ToType = "service"
	} else {
		a.clusterInfo.mu.RLock() // lock for reading
		podUid, ok := a.clusterInfo.PodIPToPodUid[skInfo.Daddr]
		a.clusterInfo.mu.RUnlock() // unlock for reading

		if ok {
			reqDto.ToUID = string(podUid)
			reqDto.ToType = "pod"
		} else {
			// 3rd party url
			if hostHeader != "" {
				reqDto.ToUID = hostHeader
				reqDto.ToType = "outbound"
			} else {
				remoteDnsHost, err := getHostnameFromIP(skInfo.Daddr)
				if err == nil {
					// dns lookup successful
					reqDto.ToUID = remoteDnsHost
					reqDto.ToType = "outbound"
				} else {
					reqDto.ToUID = skInfo.Daddr
					reqDto.ToType = "outbound"
				}
			}
		}
	}

	return nil
}

func (a *Collector) getConnKey(pid uint32, fd uint64) string {
	return fmt.Sprintf("%d-%d", pid, fd)
}

func (a *Collector) processL7(ctx context.Context, d *l7_req.L7Event) {
	// other protocols events come as whole, but http2 events come as frames
	// we need to aggregate frames to get the whole request
	defer func() {
		if r := recover(); r != nil {
			// TODO: we need to fix this properly
			logger.Logger().Debug().Msgf("probably a http2 frame sent on a closed chan: %v", r)
		}
	}()

	if d.Protocol == l7_req.L7_PROTOCOL_HTTP2 {
		var ok bool
		var ch chan *l7_req.L7Event

		a.liveProcessesMu.RLock()
		_, ok = a.liveProcesses[d.Pid]
		a.liveProcessesMu.RUnlock()
		if !ok {
			return // if a late event comes, do not create parsers and new worker to avoid memory leak
		}

		connKey := a.getConnKey(d.Pid, d.Fd)
		a.h2ParserMu.RLock()
		_, ok = a.h2Parsers[connKey]
		a.h2ParserMu.RUnlock()
		if !ok {
			// initialize parser
			a.h2ParserMu.Lock()
			a.h2Parsers[connKey] = &http2Parser{
				clientHpackDecoder: hpack.NewDecoder(4096, nil),
				serverHpackDecoder: hpack.NewDecoder(4096, nil),
			}
			a.h2ParserMu.Unlock()
		}

		a.h2ChMu.RLock()
		ch, ok = a.h2Ch[d.Pid]
		a.h2ChMu.RUnlock()
		if !ok {
			// initialize channel
			h2ChPid := make(chan *l7_req.L7Event, 100)
			a.h2ChMu.Lock()
			a.h2Ch[d.Pid] = h2ChPid
			ch = h2ChPid
			a.h2ChMu.Unlock()
			go a.processHttp2Frames(d.Pid, ch) // worker per pid, will be called once
		}

		ch <- d
		return
	}

	skInfo := a.findRelatedSocket(ctx, d)
	if skInfo == nil {
		return
	}

	// Since we process events concurrently
	// TCP events and L7 events can be processed out of order

	reqDto := models.Request{
		StartTime:  d.EventReadTime,
		Latency:    d.Duration,
		FromIP:     skInfo.Saddr,
		ToIP:       skInfo.Daddr,
		Protocol:   d.Protocol,
		Tls:        d.Tls,
		Completed:  true,
		StatusCode: d.Status,
		FailReason: "",
		Method:     d.Method,
		Tid:        d.Tid,
		Seq:        d.Seq,
	}

	if d.Protocol == l7_req.L7_PROTOCOL_POSTGRES && d.Method == l7_req.SIMPLE_QUERY {
		// parse sql command from payload
		// path = sql command
		// method = sql message type
		var err error
		reqDto.Path, err = parseSqlCommand(d.Payload[0:d.PayloadSize])
		if err != nil {
			logger.Logger().Error().AnErr("err", err)
			return
		}

	}
	var reqHostHeader string
	// parse http payload, extract path, query params, headers
	if d.Protocol == l7_req.L7_PROTOCOL_HTTP {
		_, reqDto.Path, _, reqHostHeader = parseHttpPayload(string(d.Payload[0:d.PayloadSize]))
	}

	err := a.setFromTo(skInfo, d, &reqDto, reqHostHeader)
	if err != nil {
		return
	}

	reqDto.Completed = !d.Failed

	// In AMQP-DELIVER event, we are capturing from read syscall,
	// exchange sockets
	// In Alaz context, From is always the one that makes the write
	// and To is the one that makes the read
	if d.Protocol == l7_req.L7_PROTOCOL_AMQP && d.Method == l7_req.DELIVER {
		reqDto.FromIP, reqDto.ToIP = reqDto.ToIP, reqDto.FromIP
		reqDto.FromPort, reqDto.ToPort = reqDto.ToPort, reqDto.FromPort
		reqDto.FromUID, reqDto.ToUID = reqDto.ToUID, reqDto.FromUID
		reqDto.FromType, reqDto.ToType = reqDto.ToType, reqDto.FromType
	}

	if d.Protocol == l7_req.L7_PROTOCOL_HTTP && d.Tls {
		reqDto.Protocol = "HTTPS"
	}

	err = a.eventsHandler.PersistRequest(&reqDto)
	if err != nil {
		logger.Logger().Error().Err(err).Msg("error persisting request")
	}
}

// reverse dns lookup
func getHostnameFromIP(ipAddr string) (string, error) {
	// return from cache, if exists
	// consumes too much memory otherwise
	if host, ok := reverseDnsCache.Get(ipAddr); ok {
		return host.(string), nil
	} else {
		addrs, err := net.LookupAddr(ipAddr)
		if err != nil {
			return "", err
		}

		// The reverse DNS lookup can return multiple names for the same IP.
		// In this example, we return the first name found.
		if len(addrs) > 0 {
			reverseDnsCache.Set(ipAddr, addrs[0], 0)
			return addrs[0], nil
		}
		return "", fmt.Errorf("no hostname found for IP address: %s", ipAddr)
	}
}

func (a *Collector) fetchSkLine(sockMap *SocketMap, pid uint32, fd uint64) *SocketLine {
	sockMap.mu.RLock() // lock for reading
	skLine, ok := sockMap.M[fd]
	sockMap.mu.RUnlock() // unlock for reading

	if !ok {
		logger.Logger().Debug().Uint32("pid", pid).Uint64("fd", fd).Msg("error finding skLine, go look for it")
		// start new socket line, find already established connections
		skLine = NewSocketLine(pid, fd)
		skLine.GetAlreadyExistingSockets() // find already established connections
		sockMap.mu.Lock()                  // lock for writing
		sockMap.M[fd] = skLine
		sockMap.mu.Unlock() // unlock for writing
	}

	return skLine
}

func (a *Collector) fetchSkInfo(ctx context.Context, skLine *SocketLine, d *l7_req.L7Event) *SockInfo {
	rc := retryLimit
	rt := retryInterval
	var skInfo *SockInfo
	var err error

	// skInfo, _ = skLine.GetValue(d.WriteTimeNs)

	for {
		skInfo, err = skLine.GetValue(d.WriteTimeNs)
		if err == nil && skInfo != nil {
			break
		}
		rc--
		if rc == 0 {
			break
		}
		time.Sleep(rt)
		rt *= 2 // exponential backoff

		select {
		case <-ctx.Done():
			logger.Logger().Debug().Msg("processL7 exiting, stop retrying...")
			return nil
		default:
			continue
		}
	}

	return skInfo
}

func (a *Collector) fetchSocketMap(pid uint32) *SocketMap {
	var sockMap *SocketMap
	var ok bool

	a.clusterInfo.mu.RLock() // lock for reading
	sockMap, ok = a.clusterInfo.PidToSocketMap[pid]
	a.clusterInfo.mu.RUnlock() // unlock for reading
	if !ok {
		// initialize socket map
		sockMap = &SocketMap{
			M:  make(map[uint64]*SocketLine),
			mu: sync.RWMutex{},
		}
		a.clusterInfo.mu.Lock() // lock for writing
		a.clusterInfo.PidToSocketMap[pid] = sockMap
		a.clusterInfo.mu.Unlock() // unlock for writing

		go a.ec.ListenForEncryptedReqs(pid)
	}
	return sockMap
}

func (a *Collector) findRelatedSocket(ctx context.Context, d *l7_req.L7Event) *SockInfo {
	sockMap := a.fetchSocketMap(d.Pid)
	skLine := a.fetchSkLine(sockMap, d.Pid, d.Fd)
	skInfo := a.fetchSkInfo(ctx, skLine, d)

	if skInfo == nil {
		return nil
	}

	// TODO: zero IP address check ??

	return skInfo
}

func parseSqlCommand(r []uint8) (string, error) {
	// Q, 4 bytes of length, sql command

	// skip Q, (simple query)
	r = r[1:]

	// skip 4 bytes of length
	r = r[4:]

	// get sql command
	sqlStatement := string(r)

	// garbage data can come for postgres, we need to filter out
	// search statement for sql keywords like
	if containsSQLKeywords(sqlStatement) {
		return sqlStatement, nil
	} else {
		return "", fmt.Errorf("no sql command found")
	}
}

func (a *Collector) clearSocketLines(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	skLineCh := make(chan *SocketLine, 1000)

	go func() {
		// spawn N goroutines to clear socket map
		for i := 0; i < 10; i++ {
			go func() {
				for skLine := range skLineCh {
					skLine.DeleteUnused()
				}
			}()
		}
	}()

	for range ticker.C {
		a.clusterInfo.mu.RLock()
		for _, socketMap := range a.clusterInfo.PidToSocketMap {
			for _, socketLine := range socketMap.M {
				skLineCh <- socketLine
			}
		}
		a.clusterInfo.mu.RUnlock()
	}
}
