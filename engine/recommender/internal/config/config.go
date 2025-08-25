package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"

	vpa_model "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/model"
)

// Config struct contains watcher configuration
type Configuration struct {
	Clickhouse  *Clickhouse  `toml:"clickhouse"`
	Recommender *Recommender `toml:"recommender"`
	Log         *Log         `toml:"log"`
}

type Clickhouse struct {
	Servers  []string `toml:"servers"`
	Database string   `toml:"database"`
	Username string   `toml:"username"`
	Password string   `toml:"password"`
}

type Recommender struct {
	Endpoint string `toml:"endpoint"`
	// MetricsFetcherInterval is how often metrics should be fetched
	MetricsFetcherInterval time.Duration
	// CheckpointsGCInterval is how often orphaned checkpoints should be garbage collected
	CheckpointsGCInterval time.Duration
	// Specifies storage mode. Supported values: ClickHouse, checkpoint (default)
	Storage string
	// HistoryLength is how much time back ClickHouse have to be queried to get historical metrics
	HistoryLength string
	// HistoryResolution is the resolution at which ClickHouse is queried for historical metrics
	HistoryResolution string
	// QueryTimeout is the how long to wait before killing long queries
	QueryTimeout string
	// MemoryAggregationInterval is the length of a single interval, for which the peak memory usage is computed. Memory usage peaks are aggregated in multiples of this interval. In other words there is one memory usage sample per interval (the maximum usage over that interval)
	MemoryAggregationInterval time.Duration
	// MemoryAggregationIntervalCount is the number of consecutive memory-aggregation-intervals which make up the MemoryAggregationWindowLength which in turn is the period for memory usage aggregation by VPA. In other words, MemoryAggregationWindowLength = memory-aggregation-interval * memory-aggregation-interval-count.
	MemoryAggregationIntervalCount int64
	// MemoryHistogramDecayHalfLife is the amount of time it takes a historical memory usage sample to lose half of its weight. In other words, a fresh usage sample is twice as 'important' as one with age equal to the half life period.
	MemoryHistogramDecayHalfLife time.Duration
	// CPUHistogramDecayHalfLife is the amount of time it takes a historical CPU usage sample to lose half of its weight.
	CPUHistogramDecayHalfLife time.Duration
	// horizontalPodAutoscalerSyncPeriod is the period for syncing the number of pods in MPA.
	HPASyncPeriod time.Duration
	// horizontalPodAutoscalerUpscaleForbiddenWindow is a period after which next upscale allowed.
	HPAUpscaleForbiddenWindow time.Duration
	// horizontalPodAutoscalerDownscaleForbiddenWindow is a period after which next downscale allowed.
	HPADownscaleForbiddenWindow time.Duration
	// HorizontalPodAutoscalerDowncaleStabilizationWindow is a period for which autoscaler will look
	// backwards and not scale down below any recommendation it made during that period.
	HPADownscaleStabilizationWindow time.Duration
	// horizontalPodAutoscalerTolerance is the tolerance for when resource usage suggests upscaling/downscaling
	HPATolerance float64
	// HorizontalPodAutoscalerCPUInitializationPeriod is the period after pod start when CPU samples
	// might be skipped.
	HPACPUInitializationPeriod time.Duration
	// HorizontalPodAutoscalerInitialReadinessDelay is period after pod start during which readiness
	// changes are treated as readiness being set for the first time. The only effect of this is
	// that HPA will disregard CPU samples from unready pods that had last readiness change during
	// that period.
	HPAInitialReadinessDelay time.Duration
	ConcurrentHPASyncs       int64
}

type Log struct {
	Level string `toml:"level"`
}

func Fetch() *Configuration {
	return &Configuration{
		Clickhouse: &Clickhouse{
			Servers:  viper.GetStringSlice("clickhouse.servers"),
			Database: viper.GetString("clickhouse.database"),
			Username: viper.GetString("clickhouse.username"),
			Password: viper.GetString("clickhouse.password"),
		},
		Recommender: &Recommender{
			Endpoint:                        getStringOrDefault("recommender.endpoint", "localhost:50052"),
			MetricsFetcherInterval:          getDurationOrDefault("recommender-interval", 1*time.Minute),
			CheckpointsGCInterval:           getDurationOrDefault("checkpoints-gc-interval", 10*time.Minute),
			Storage:                         getStringOrDefault("storage", "checkpoint"),
			HistoryLength:                   getStringOrDefault("history-length", "8d"),
			HistoryResolution:               getStringOrDefault("history-resolution", "1h"),
			QueryTimeout:                    getStringOrDefault("clickhouse-query-timeout", "5m"),
			MemoryAggregationInterval:       getDurationOrDefault("memory-aggregation-interval", vpa_model.DefaultMemoryAggregationInterval),
			MemoryAggregationIntervalCount:  getInt64OrDefault("memory-aggregation-interval-count", vpa_model.DefaultMemoryAggregationIntervalCount),
			MemoryHistogramDecayHalfLife:    getDurationOrDefault("memory-histogram-decay-half-life", vpa_model.DefaultMemoryHistogramDecayHalfLife),
			CPUHistogramDecayHalfLife:       getDurationOrDefault("cpu-histogram-decay-half-life", vpa_model.DefaultCPUHistogramDecayHalfLife),
			HPASyncPeriod:                   getDurationOrDefault("hpa-sync-period", 15*time.Second),
			HPAUpscaleForbiddenWindow:       getDurationOrDefault("hpa-upscale-forbidden-window", 3*time.Minute),
			HPADownscaleForbiddenWindow:     getDurationOrDefault("hpa-downscale-forbidden-window", 5*time.Minute),
			HPADownscaleStabilizationWindow: getDurationOrDefault("hpa-downscale-stabilization-window", 5*time.Minute),
			HPATolerance:                    getFloat64OrDefault("hpa-tolerance", 0.1),
			HPACPUInitializationPeriod:      getDurationOrDefault("hpa-cpu-initialization-period", 5*time.Minute),
			HPAInitialReadinessDelay:        getDurationOrDefault("hpa-initial-readiness-delay", 30*time.Second),
			ConcurrentHPASyncs:              getInt64OrDefault("concurrent-hpa-syncs", 5),
		},
		Log: &Log{
			Level: getStringOrDefault("log.level", "info"),
		},
	}
}

func (c *Configuration) Validate() error {
	if c.Recommender == nil {
		return errors.New("[recommender] entry is missing from the configuration")
	}
	if c.Clickhouse == nil {
		return errors.New("[clickhouse] entry is missing from the configuration")
	}
	if len(c.Clickhouse.Servers) == 0 {
		return errors.New("clickhouse server endpoint missing from the list")
	}
	if c.Clickhouse.Username == "" {
		return errors.New("clickhouse username is missing")
	}
	if c.Clickhouse.Password == "" {
		return errors.New("clickhouse password is missing")
	}
	if c.Clickhouse.Database == "" {
		return errors.New("clickhouse database is missing")
	}
	if c.Log != nil {
		if c.Log.Level != "debug" && c.Log.Level != "info" && c.Log.Level != "warn" && c.Log.Level != "error" {
			return errors.New("invalid log level value")
		}
	}
	return nil
}

func getStringOrDefault(key string, defaultValue string) string {
	value := viper.GetString(key)
	if value != "" {
		return value
	}
	return defaultValue
}

func getDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	value := viper.GetDuration(key)
	if value != time.Duration(0) {
		return value
	}
	return defaultValue
}

func getInt64OrDefault(key string, defaultValue int64) int64 {
	value := viper.GetInt64(key)
	if value != 0 {
		return value
	}
	return defaultValue
}

func getFloat64OrDefault(key string, defaultValue float64) float64 {
	value := viper.GetFloat64(key)
	if value != 0 {
		return value
	}
	return defaultValue
}
