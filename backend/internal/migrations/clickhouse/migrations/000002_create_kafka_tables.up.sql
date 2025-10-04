CREATE TABLE
    IF NOT EXISTS kubernetes_events_kafka (
        organization_id UInt32,
        cluster_id Int64,
        k8s_version String,
        idempotency_key String,
        watcher_version String,
        received_at_unix Int64,
        object_kind String,
        object_name String,
        object_uid String,
        object_fieldpath String,
        object_timestamp Int64,
        object_namespace String,
        event_severity_number Int64,
        event_severity_text String,
        event_reason String,
        event_action String,
        event_starttime String,
        event_name String,
        event_message String,
        event_uid String,
        event_count Int64,
        object_api_version String,
        object_resource_version String,
        node_name String
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092', -- needs to be templetized
    kafka_topic_list = 'store.k8s.events', -- needs to be templetized
    kafka_group_name = 'clickhouse-kubernetes-consumer', -- needs to be templetized
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1, -- needs to be templetized
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubernetes_events TO kubernetes_events AS
SELECT
    *
FROM
    kubernetes_events_kafka;

-- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
CREATE TABLE
    IF NOT EXISTS kubernetes_objects_kafka (
        organization_id UInt32,
        cluster_id Int64,
        cloud_provider String,
        k8s_version String,
        idempotency_key String,
        received_at_unix Int64,
        timestamp Int64,
        type String,
        kind String,
        namespace String,
        name String,
        uid String,
        labels Map (String, String),
        annotations Map (String, String),
        node_name String,
        status String,
        phase String,
        restart_count Int64,
        owner_kind String,
        owner_name String,
        replicas Int64,
        ready_replicas Int64,
        available_replicas Int64,
        updated_replicas Int64,
        current_replicas Int64,
        tolerations Array (String),
        affinity Map (String, String),
        extra Map (String, String),
        api_version String,
        hpa_max_replicas Int64,
        hpa_min_replicas Int64,
        hpa_scale_target_ref String,
        job_active Int64,
        job_failed Int64,
        job_succeeded Int64,
        job_parallelism Int64,
        job_completions Int64,
        ns_phase_value Int64,
        kubelet_version String,
        os_type String,
        os_image String,
        container_runtime String,
        container_runtime_version String,
        pod_reason String,
        qos_class String,
        daemonset_current_number_scheduled Int64,
        daemonset_desired_number_scheduled Int64,
        daemonset_number_misscheduled Int64,
        daemonset_number_ready Int64,
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.k8s.objects',
    kafka_group_name = 'k8s_objects_consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubernetes_objects TO kubernetes_objects AS
SELECT
    *
FROM
    kubernetes_objects_kafka;

CREATE TABLE
    IF NOT EXISTS kubernetes_containers_kafka (
        uid String,
        idempotency_key String,
        timestamp Int64,
        container_name String,
        image String,
        image_tag String,
        container_id String,
        restarts_count Int64,
        ready Int64,
        state_type String,
        state_json String,
        cpu_limits String,
        memory_limits String,
        cpu_requests String,
        memory_requests String
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.k8s.containers',
    kafka_group_name = 'k8s_containers_consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubernetes_containers TO kubernetes_containers AS
SELECT
    *
FROM
    kubernetes_containers_kafka;

CREATE TABLE
    IF NOT EXISTS kubernetes_volumes_kafka (
        uid String,
        idempotency_key String,
        timestamp Int64,
        volume_name String,
        volume_type String
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.k8s.volumes',
    kafka_group_name = 'k8s_volumes_consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubernetes_volumes TO kubernetes_volumes AS
SELECT
    *
FROM
    kubernetes_volumes_kafka;

CREATE TABLE
    IF NOT EXISTS kubernetes_node_conditions_kafka (
        uid String,
        idempotency_key String,
        timestamp Int64,
        condition_type String,
        condition_status String,
        reason String,
        message String
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.k8s.node_conditions',
    kafka_group_name = 'k8s_nodeconditions_consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubernetes_node_conditions TO kubernetes_node_conditions AS
SELECT
    *
FROM
    kubernetes_node_conditions_kafka;

CREATE TABLE
    IF NOT EXISTS kubernetes_allocatable_resources_kafka (
        uid String,
        idempotency_key String,
        timestamp Int64,
        cpu String,
        memory String,
        pods String,
        ephemeral_storage String,
        others Map (String, String)
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.k8s.allocatable_resources',
    kafka_group_name = 'k8s_allocatable_consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubernetes_allocatable_resources TO kubernetes_allocatable_resources AS
SELECT
    *
FROM
    kubernetes_allocatable_resources_kafka;

CREATE TABLE
    IF NOT EXISTS kubernetes_cluster_quotas_kafka (
        uid String,
        idempotency_key String,
        timestamp Int64,
        quota_name String,
        quota_uid String,
        total_limits Array (Tuple (String, Int64)),
        total_usage Array (Tuple (String, Int64))
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.k8s.cluster_quotas',
    kafka_group_name = 'k8s_clusterquotas_consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubernetes_cluster_quotas TO kubernetes_cluster_quotas AS
SELECT
    *
FROM
    kubernetes_cluster_quotas_kafka;

CREATE TABLE
    IF NOT EXISTS kubernetes_namespace_quotas_kafka (
        uid String,
        idempotency_key String,
        timestamp Int64,
        namespace String,
        limits Array (Tuple (String, Int64)),
        usage Array (Tuple (String, Int64))
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.k8s.namespace_quotas',
    kafka_group_name = 'k8s_namespacequotas_consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubernetes_namespace_quotas TO kubernetes_namespace_quotas AS
SELECT
    *
FROM
    kubernetes_namespace_quotas_kafka;

-- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
CREATE TABLE
    IF NOT EXISTS kubelet_node_metrics_kafka (
        organization_id UInt32,
        idempotency_key String,
        cluster_id Int64,
        received_at_unix Int64,
        timestamp Int64,
        node_name String,
        start_time Int64,
        cpu_usage_nano_cores UInt64,
        cpu_usage_core_nanoseconds UInt64,
        cpu_psi_full_avg10 Float64,
        cpu_psi_some_avg10 Float64,
        memory_available_bytes UInt64,
        memory_usage_bytes UInt64,
        memory_working_set_bytes UInt64,
        memory_rss_bytes UInt64,
        memory_page_faults UInt64,
        memory_major_page_faults UInt64,
        memory_psi_full_avg10 Float64,
        memory_psi_some_avg10 Float64,
        io_psi_full_avg10 Float64,
        io_psi_some_avg10 Float64,
        fs_available_bytes UInt64,
        fs_capacity_bytes UInt64,
        fs_used_bytes UInt64,
        swap_available_bytes UInt64,
        swap_usage_bytes UInt64
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.kubelet.node.metrics',
    kafka_group_name = 'clickhouse-kubelet-node-consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubelet_node_metrics TO kubelet_node_metrics AS
SELECT
    *
FROM
    kubelet_node_metrics_kafka;

CREATE TABLE
    IF NOT EXISTS kubelet_pod_metrics_kafka (
        timestamp Int64,
        idempotency_key String,
        pod_uid String,
        pod_name String,
        pod_namespace String,
        start_time Int64,
        cpu_usage_nano_cores UInt64,
        memory_usage_bytes UInt64,
        memory_working_set_bytes UInt64,
        network_rx_bytes UInt64,
        network_tx_bytes UInt64,
        process_count UInt64,
        swap_available_bytes UInt64,
        swap_usage_bytes UInt64
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.kubelet.pod.metrics',
    kafka_group_name = 'clickhouse-kubelet-pod-consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubelet_pod_metrics TO kubelet_pod_metrics AS
SELECT
    *
FROM
    kubelet_pod_metrics_kafka;

CREATE TABLE
    IF NOT EXISTS kubelet_container_metrics_kafka (
        timestamp Int64,
        idempotency_key String,
        pod_uid String,
        container_name String,
        container_start_time Int64,
        cpu_usage_nano_cores UInt64,
        memory_usage_bytes UInt64,
        memory_working_set_bytes UInt64,
        swap_available_bytes UInt64,
        swap_usage_bytes UInt64,
        rootfs_used_bytes UInt64,
        logs_used_bytes UInt64,
        accelerator JSON,
        user_metrics JSON
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.kubelet.container.metrics',
    kafka_group_name = 'clickhouse-kubelet-container-consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubelet_container_metrics TO kubelet_container_metrics AS
SELECT
    *
FROM
    kubelet_container_metrics_kafka;

CREATE TABLE
    IF NOT EXISTS kubelet_network_metrics_kafka (
        timestamp Int64,
        idempotency_key String,
        pod_uid String,
        interface_name String,
        rx_bytes UInt64,
        rx_errors UInt64,
        tx_bytes UInt64,
        tx_errors UInt64
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.kubelet.network.metrics',
    kafka_group_name = 'clickhouse-kubelet-network-consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubelet_network_metrics TO kubelet_network_metrics AS
SELECT
    *
FROM
    kubelet_network_metrics_kafka;

CREATE TABLE
    IF NOT EXISTS kubelet_volume_metrics_kafka (
        timestamp Int64,
        idempotency_key String,
        pod_uid String,
        volume_name String,
        pvc_name String,
        pvc_namespace String,
        abnormal Bool,
        available_bytes UInt64,
        capacity_bytes UInt64,
        used_bytes UInt64,
        inodes_free UInt64,
        inodes UInt64,
        inodes_used UInt64
    ) ENGINE = Kafka SETTINGS kafka_broker_list = 'kafka1:9092',
    kafka_topic_list = 'store.kubelet.volume.metrics',
    kafka_group_name = 'clickhouse-kubelet-volume-consumer',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_thread_per_consumer = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kubelet_volume_metrics TO kubelet_volume_metrics AS
SELECT
    *
FROM
    kubelet_volume_metrics_kafka;