CREATE TABLE
    IF NOT EXISTS kubernetes_objects (
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
        cluster_quota_name String,
        cluster_quota_uid String,
        daemonset_current_number_scheduled Int64,
        daemonset_desired_number_scheduled Int64,
        daemonset_number_misscheduled Int64,
        daemonset_number_ready Int64
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