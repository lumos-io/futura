-- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
CREATE TABLE
    kubernetes_events (
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
        event_starttime String, -- Consider converting to DateTime64 if needed
        event_name String,
        event_message String,
        event_uid String,
        event_count Int64,
        object_api_version String,
        object_resource_version String,
        node_name String
    ) ENGINE = MergeTree
PARTITION BY
    toDate (object_timestamp)
ORDER BY
    (
        organization_id,
        cluster_id,
        object_namespace,
        object_kind,
        object_name,
        object_timestamp
    );

-- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
CREATE TABLE
    IF NOT EXISTS kubernetes_objects (
        organization_id UInt32,
        cluster_id Int64,
        k8s_version String,
        received_at_unix Int64,
        timestamp DateTime64 (3),
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
        idempotency_key String,
        watcher_version String,
        cloud_provider String,
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (kind, namespace, name, timestamp);

CREATE TABLE
    IF NOT EXISTS kubernetes_containers (
        uid String,
        timestamp DateTime64 (3),
        container_name String,
        image String,
        image_tag String,
        container_id String,
        restarts_count Int64,
        ready Int64,
        state_type Enum8 ('waiting' = 1, 'running' = 2, 'terminated' = 3),
        state_json String,
        cpu_limits String,
        memory_limits String,
        cpu_requests String,
        memory_requests String
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (uid, container_name, timestamp);

CREATE TABLE
    IF NOT EXISTS kubernetes_volumes (
        uid String,
        timestamp DateTime64 (3),
        volume_name String,
        volume_type String
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (uid, volume_name);

CREATE TABLE
    IF NOT EXISTS kubernetes_node_conditions (
        uid String,
        timestamp DateTime64 (3),
        condition_type String,
        condition_status String,
        reason String,
        message String
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (uid, condition_type);

CREATE TABLE
    IF NOT EXISTS kubernetes_allocatable_resources (
        uid String,
        timestamp DateTime64 (3),
        cpu String,
        memory String,
        pods String,
        ephemeral_storage String,
        others Map (String, String)
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    uid;

CREATE TABLE
    IF NOT EXISTS kubernetes_cluster_quotas (
        uid String,
        timestamp DateTime64 (3),
        quota_name String,
        quota_uid String,
        total_limits Array (Tuple (String, Int64)),
        total_usage Array (Tuple (String, Int64))
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (uid, quota_name);

CREATE TABLE
    IF NOT EXISTS kubernetes_namespace_quotas (
        uid String,
        timestamp DateTime64 (3),
        namespace String,
        limits Array (Tuple (String, Int64)),
        usage Array (Tuple (String, Int64))
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (uid, namespace);

-- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
CREATE TABLE
    kubelet_node_metrics (
        organization_id UInt32,
        cluster_id Int64,
        received_at_unix Int64,
        timestamp DateTime64 (3),
        node_name String,
        start_time DateTime64 (3),
        -- CPU
        cpu_usage_nano_cores UInt64,
        cpu_usage_core_nanoseconds UInt64,
        cpu_psi_full_avg10 Float64,
        cpu_psi_some_avg10 Float64,
        -- Memory
        memory_available_bytes UInt64,
        memory_usage_bytes UInt64,
        memory_working_set_bytes UInt64,
        memory_rss_bytes UInt64,
        memory_page_faults UInt64,
        memory_major_page_faults UInt64,
        memory_psi_full_avg10 Float64,
        memory_psi_some_avg10 Float64,
        -- IO
        io_psi_full_avg10 Float64,
        io_psi_some_avg10 Float64,
        -- FS
        fs_available_bytes UInt64,
        fs_capacity_bytes UInt64,
        fs_used_bytes UInt64,
        -- Swap
        swap_available_bytes UInt64,
        swap_usage_bytes UInt64
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (node_name, timestamp);

CREATE TABLE
    kubelet_pod_metrics (
        timestamp DateTime64 (3),
        pod_uid String,
        pod_name String,
        pod_namespace String,
        start_time DateTime64 (3),
        cpu_usage_nano_cores UInt64,
        memory_usage_bytes UInt64,
        memory_working_set_bytes UInt64,
        network_rx_bytes UInt64,
        network_tx_bytes UInt64,
        process_count UInt64,
        swap_available_bytes UInt64,
        swap_usage_bytes UInt64
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (pod_namespace, pod_name, timestamp);

CREATE TABLE
    kubelet_container_metrics (
        timestamp DateTime64 (3),
        pod_uid String,
        container_name String,
        container_start_time DateTime64 (3),
        cpu_usage_nano_cores UInt64,
        memory_usage_bytes UInt64,
        memory_working_set_bytes UInt64,
        swap_available_bytes UInt64,
        swap_usage_bytes UInt64,
        rootfs_used_bytes UInt64,
        logs_used_bytes UInt64,
        accelerator JSON, -- or flatten if you have predictable models
        user_metrics JSON
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (pod_uid, container_name, timestamp);

CREATE TABLE
    kubelet_network_metrics (
        timestamp DateTime64 (3),
        pod_uid String,
        interface_name String,
        rx_bytes UInt64,
        rx_errors UInt64,
        tx_bytes UInt64,
        tx_errors UInt64
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (pod_uid, interface_name, timestamp);

CREATE TABLE
    kubelet_volume_metrics (
        timestamp DateTime64 (3),
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
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (timestamp)
ORDER BY
    (pod_uid, volume_name, timestamp);