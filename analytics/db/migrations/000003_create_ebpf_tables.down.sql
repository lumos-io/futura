-- Drop Materialized View
DROP VIEW IF EXISTS mv_ebpf_metrics;

-- Drop Kafka Consumer Table
DROP TABLE IF EXISTS ebpf_metrics_kafka;

-- Drop Main eBPF Metrics Table
DROP TABLE IF EXISTS ebpf_metrics;