CREATE TABLE
    IF NOT EXISTS recommendation_decisions (
        cluster_id String,
        namespace String,
        app_name String,
        workload_kind String,
        decision_id String,
        model_version String,
        confidence Float64,
        audit_reasons Array (String),
        plan_vertical Array (
            Nested (
                container String,
                resource String,
                value String -- Kubernetes quantity string
            )
        ),
        plan_replicas Int32,
        effective_policy Map (String, String), -- JSON or kv of resource bounds
        ts DateTime64 (3) DEFAULT now ()
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (ts)
ORDER BY
    (cluster_id, namespace, app_name, ts);

CREATE TABLE
    IF NOT EXISTS execution_outcomes (
        cluster_id String,
        namespace String,
        app_name String,
        workload_kind String,
        decision_id String,
        success UInt8,
        note String,
        post_action_metrics Map (String, Float64), -- snapshot of key metrics
        reported_at DateTime64 (3),
        ts DateTime64 (3) DEFAULT now ()
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (ts)
ORDER BY
    (cluster_id, namespace, app_name, reported_at);

CREATE TABLE
    IF NOT EXISTS model_registry (
        cluster_id String,
        namespace String,
        app_name String,
        workload_kind String,
        model_version String,
        policy_name String,
        checkpoint_uri String,
        labels Map (String, String),
        compatible_feature_schema Array (String),
        updated_at DateTime64 (3),
        ts DateTime64 (3) DEFAULT now ()
    ) ENGINE = ReplacingMergeTree (updated_at)
PARTITION BY
    toYYYYMM (ts)
ORDER BY
    (cluster_id, namespace, app_name, model_version);

CREATE TABLE
    IF NOT EXISTS training_jobs (
        training_id String,
        cluster_id String,
        namespace String,
        app_name String,
        workload_kind String,
        reason String, -- "scheduled","drift","manual"
        horizon_hours Int64,
        base_version String,
        hparams Map (String, String),
        job_name String, -- Kubernetes job name (if used)
        status String, -- "PENDING","RUNNING","COMPLETED","FAILED","CANCELLED"
        error_message String,
        created_at DateTime64 (3) DEFAULT now (),
        updated_at DateTime64 (3)
    ) ENGINE = ReplacingMergeTree (updated_at)
PARTITION BY
    toYYYYMM (created_at)
ORDER BY
    (training_id, created_at);

CREATE TABLE
    IF NOT EXISTS training_progress (
        training_id String,
        agent_id String,
        step Int32,
        train_loss Float64,
        eval_reward Float64,
        progress_pct Float64,
        scalars Map (String, Float64),
        ts DateTime64 (3) DEFAULT now ()
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (ts)
ORDER BY
    (training_id, agent_id, step, ts);

CREATE TABLE
    IF NOT EXISTS training_results (
        training_id String,
        agent_id String,
        success UInt8,
        error_message String,
        model_version String,
        checkpoint_uri String,
        eval_reward Float64,
        metrics Map (String, Float64),
        ts DateTime64 (3) DEFAULT now ()
    ) ENGINE = ReplacingMergeTree (ts)
PARTITION BY
    toYYYYMM (ts)
ORDER BY
    (training_id, agent_id);

CREATE TABLE
    IF NOT EXISTS agent_heartbeats (
        training_id String,
        agent_id String,
        sysinfo Map (String, String), -- gpu/mem info
        ts DateTime64 (3) DEFAULT now ()
    ) ENGINE = MergeTree
PARTITION BY
    toYYYYMM (ts)
ORDER BY
    (training_id, agent_id, ts);