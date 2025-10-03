import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkloadKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKLOAD_KIND_UNSPECIFIED: _ClassVar[WorkloadKind]
    DEPLOYMENT: _ClassVar[WorkloadKind]
    STATEFUL_SET: _ClassVar[WorkloadKind]
    DAEMON_SET: _ClassVar[WorkloadKind]
    JOB: _ClassVar[WorkloadKind]
    CRON_JOB: _ClassVar[WorkloadKind]
WORKLOAD_KIND_UNSPECIFIED: WorkloadKind
DEPLOYMENT: WorkloadKind
STATEFUL_SET: WorkloadKind
DAEMON_SET: WorkloadKind
JOB: WorkloadKind
CRON_JOB: WorkloadKind

class ClusterOptimizationConfigRequest(_message.Message):
    __slots__ = ("api_key", "cloud_provider", "region", "cost_sensitivity", "monthly_budget", "preferred_instance_types", "allow_spot", "max_spot_percentage", "cluster_scaling_mode")
    API_KEY_FIELD_NUMBER: _ClassVar[int]
    CLOUD_PROVIDER_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    COST_SENSITIVITY_FIELD_NUMBER: _ClassVar[int]
    MONTHLY_BUDGET_FIELD_NUMBER: _ClassVar[int]
    PREFERRED_INSTANCE_TYPES_FIELD_NUMBER: _ClassVar[int]
    ALLOW_SPOT_FIELD_NUMBER: _ClassVar[int]
    MAX_SPOT_PERCENTAGE_FIELD_NUMBER: _ClassVar[int]
    CLUSTER_SCALING_MODE_FIELD_NUMBER: _ClassVar[int]
    api_key: str
    cloud_provider: str
    region: str
    cost_sensitivity: str
    monthly_budget: str
    preferred_instance_types: _containers.RepeatedScalarFieldContainer[str]
    allow_spot: bool
    max_spot_percentage: str
    cluster_scaling_mode: str
    def __init__(self, api_key: _Optional[str] = ..., cloud_provider: _Optional[str] = ..., region: _Optional[str] = ..., cost_sensitivity: _Optional[str] = ..., monthly_budget: _Optional[str] = ..., preferred_instance_types: _Optional[_Iterable[str]] = ..., allow_spot: bool = ..., max_spot_percentage: _Optional[str] = ..., cluster_scaling_mode: _Optional[str] = ...) -> None: ...

class ClusterOptimizationConfigResponse(_message.Message):
    __slots__ = ("success", "message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    def __init__(self, success: bool = ..., message: _Optional[str] = ...) -> None: ...

class SyncSLORequest(_message.Message):
    __slots__ = ("api_key", "service_name", "target_p95_latency", "target_error_rate", "target_throughput", "priority", "last_updated")
    API_KEY_FIELD_NUMBER: _ClassVar[int]
    SERVICE_NAME_FIELD_NUMBER: _ClassVar[int]
    TARGET_P95_LATENCY_FIELD_NUMBER: _ClassVar[int]
    TARGET_ERROR_RATE_FIELD_NUMBER: _ClassVar[int]
    TARGET_THROUGHPUT_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATED_FIELD_NUMBER: _ClassVar[int]
    api_key: str
    service_name: str
    target_p95_latency: str
    target_error_rate: str
    target_throughput: str
    priority: str
    last_updated: _timestamp_pb2.Timestamp
    def __init__(self, api_key: _Optional[str] = ..., service_name: _Optional[str] = ..., target_p95_latency: _Optional[str] = ..., target_error_rate: _Optional[str] = ..., target_throughput: _Optional[str] = ..., priority: _Optional[str] = ..., last_updated: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SyncSLOResponse(_message.Message):
    __slots__ = ("success", "message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    def __init__(self, success: bool = ..., message: _Optional[str] = ...) -> None: ...

class AppRef(_message.Message):
    __slots__ = ("api_key", "namespace", "app_name", "kind")
    API_KEY_FIELD_NUMBER: _ClassVar[int]
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    APP_NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    api_key: str
    namespace: str
    app_name: str
    kind: WorkloadKind
    def __init__(self, api_key: _Optional[str] = ..., namespace: _Optional[str] = ..., app_name: _Optional[str] = ..., kind: _Optional[_Union[WorkloadKind, str]] = ...) -> None: ...

class ClusterRef(_message.Message):
    __slots__ = ("api_key",)
    API_KEY_FIELD_NUMBER: _ClassVar[int]
    api_key: str
    def __init__(self, api_key: _Optional[str] = ...) -> None: ...

class ResourceLimit(_message.Message):
    __slots__ = ("min", "max")
    MIN_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    min: str
    max: str
    def __init__(self, min: _Optional[str] = ..., max: _Optional[str] = ...) -> None: ...

class SafetyPolicy(_message.Message):
    __slots__ = ("resource_bounds", "max_increase_ratio", "max_decrease_ratio", "min_action_cooldown_seconds")
    class ResourceBoundsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ResourceLimit
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ResourceLimit, _Mapping]] = ...) -> None: ...
    RESOURCE_BOUNDS_FIELD_NUMBER: _ClassVar[int]
    MAX_INCREASE_RATIO_FIELD_NUMBER: _ClassVar[int]
    MAX_DECREASE_RATIO_FIELD_NUMBER: _ClassVar[int]
    MIN_ACTION_COOLDOWN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    resource_bounds: _containers.MessageMap[str, ResourceLimit]
    max_increase_ratio: float
    max_decrease_ratio: float
    min_action_cooldown_seconds: int
    def __init__(self, resource_bounds: _Optional[_Mapping[str, ResourceLimit]] = ..., max_increase_ratio: _Optional[float] = ..., max_decrease_ratio: _Optional[float] = ..., min_action_cooldown_seconds: _Optional[int] = ...) -> None: ...

class CandidateProposal(_message.Message):
    __slots__ = ("container", "resources", "target_replicas", "source")
    class ResourcesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CONTAINER_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    TARGET_REPLICAS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    container: str
    resources: _containers.ScalarMap[str, str]
    target_replicas: int
    source: str
    def __init__(self, container: _Optional[str] = ..., resources: _Optional[_Mapping[str, str]] = ..., target_replicas: _Optional[int] = ..., source: _Optional[str] = ...) -> None: ...

class ContainerPatch(_message.Message):
    __slots__ = ("container_name", "cpu_95th_nano", "memory_95th_bytes", "recommended_cpu_nano", "recommended_memory_bytes", "current_cpu_request_nano", "current_memory_request_bytes", "recommendation_notes")
    CONTAINER_NAME_FIELD_NUMBER: _ClassVar[int]
    CPU_95TH_NANO_FIELD_NUMBER: _ClassVar[int]
    MEMORY_95TH_BYTES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_CPU_NANO_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_MEMORY_BYTES_FIELD_NUMBER: _ClassVar[int]
    CURRENT_CPU_REQUEST_NANO_FIELD_NUMBER: _ClassVar[int]
    CURRENT_MEMORY_REQUEST_BYTES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_NOTES_FIELD_NUMBER: _ClassVar[int]
    container_name: str
    cpu_95th_nano: int
    memory_95th_bytes: int
    recommended_cpu_nano: int
    recommended_memory_bytes: int
    current_cpu_request_nano: int
    current_memory_request_bytes: int
    recommendation_notes: str
    def __init__(self, container_name: _Optional[str] = ..., cpu_95th_nano: _Optional[int] = ..., memory_95th_bytes: _Optional[int] = ..., recommended_cpu_nano: _Optional[int] = ..., recommended_memory_bytes: _Optional[int] = ..., current_cpu_request_nano: _Optional[int] = ..., current_memory_request_bytes: _Optional[int] = ..., recommendation_notes: _Optional[str] = ...) -> None: ...

class HpaScaleAction(_message.Message):
    __slots__ = ("replicas",)
    REPLICAS_FIELD_NUMBER: _ClassVar[int]
    replicas: int
    def __init__(self, replicas: _Optional[int] = ...) -> None: ...

class ClusterProvisionAction(_message.Message):
    __slots__ = ("node_groups", "strategy", "bin_packing_strategy")
    NODE_GROUPS_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    BIN_PACKING_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    node_groups: _containers.RepeatedCompositeFieldContainer[NodeGroupProvision]
    strategy: str
    bin_packing_strategy: str
    def __init__(self, node_groups: _Optional[_Iterable[_Union[NodeGroupProvision, _Mapping]]] = ..., strategy: _Optional[str] = ..., bin_packing_strategy: _Optional[str] = ...) -> None: ...

class NodeGroupProvision(_message.Message):
    __slots__ = ("name", "instance_types", "count", "capacity_type", "availability_zone", "labels", "taints", "reason", "target_workloads")
    class LabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_TYPES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_TYPE_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_ZONE_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    TAINTS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    TARGET_WORKLOADS_FIELD_NUMBER: _ClassVar[int]
    name: str
    instance_types: _containers.RepeatedScalarFieldContainer[str]
    count: int
    capacity_type: str
    availability_zone: str
    labels: _containers.ScalarMap[str, str]
    taints: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    target_workloads: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., instance_types: _Optional[_Iterable[str]] = ..., count: _Optional[int] = ..., capacity_type: _Optional[str] = ..., availability_zone: _Optional[str] = ..., labels: _Optional[_Mapping[str, str]] = ..., taints: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ..., target_workloads: _Optional[_Iterable[str]] = ...) -> None: ...

class VpaRecommendAction(_message.Message):
    __slots__ = ("container", "cpu_request_mcpu", "memory_mib", "mode")
    CONTAINER_FIELD_NUMBER: _ClassVar[int]
    CPU_REQUEST_MCPU_FIELD_NUMBER: _ClassVar[int]
    MEMORY_MIB_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    container: str
    cpu_request_mcpu: int
    memory_mib: int
    mode: str
    def __init__(self, container: _Optional[str] = ..., cpu_request_mcpu: _Optional[int] = ..., memory_mib: _Optional[int] = ..., mode: _Optional[str] = ...) -> None: ...

class AppActionPlan(_message.Message):
    __slots__ = ("type", "confidence", "reason", "hpa_scale", "vpa_recommend")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    HPA_SCALE_FIELD_NUMBER: _ClassVar[int]
    VPA_RECOMMEND_FIELD_NUMBER: _ClassVar[int]
    type: str
    confidence: float
    reason: str
    hpa_scale: HpaScaleAction
    vpa_recommend: VpaRecommendAction
    def __init__(self, type: _Optional[str] = ..., confidence: _Optional[float] = ..., reason: _Optional[str] = ..., hpa_scale: _Optional[_Union[HpaScaleAction, _Mapping]] = ..., vpa_recommend: _Optional[_Union[VpaRecommendAction, _Mapping]] = ...) -> None: ...

class ClusterActionPlan(_message.Message):
    __slots__ = ("action_type", "confidence", "reason", "provision", "deprovision", "no_action", "cost_benefit", "urgency", "execute_within_seconds")
    ACTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    PROVISION_FIELD_NUMBER: _ClassVar[int]
    DEPROVISION_FIELD_NUMBER: _ClassVar[int]
    NO_ACTION_FIELD_NUMBER: _ClassVar[int]
    COST_BENEFIT_FIELD_NUMBER: _ClassVar[int]
    URGENCY_FIELD_NUMBER: _ClassVar[int]
    EXECUTE_WITHIN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    action_type: str
    confidence: float
    reason: str
    provision: ClusterProvisionAction
    deprovision: ClusterDeprovisionAction
    no_action: ClusterNoAction
    cost_benefit: CostBenefit
    urgency: float
    execute_within_seconds: int
    def __init__(self, action_type: _Optional[str] = ..., confidence: _Optional[float] = ..., reason: _Optional[str] = ..., provision: _Optional[_Union[ClusterProvisionAction, _Mapping]] = ..., deprovision: _Optional[_Union[ClusterDeprovisionAction, _Mapping]] = ..., no_action: _Optional[_Union[ClusterNoAction, _Mapping]] = ..., cost_benefit: _Optional[_Union[CostBenefit, _Mapping]] = ..., urgency: _Optional[float] = ..., execute_within_seconds: _Optional[int] = ...) -> None: ...

class ClusterDeprovisionAction(_message.Message):
    __slots__ = ("node_names", "strategy", "max_parallel", "drain_timeout_seconds", "reason")
    NODE_NAMES_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    MAX_PARALLEL_FIELD_NUMBER: _ClassVar[int]
    DRAIN_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    node_names: _containers.RepeatedScalarFieldContainer[str]
    strategy: str
    max_parallel: int
    drain_timeout_seconds: int
    reason: str
    def __init__(self, node_names: _Optional[_Iterable[str]] = ..., strategy: _Optional[str] = ..., max_parallel: _Optional[int] = ..., drain_timeout_seconds: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class ClusterNoAction(_message.Message):
    __slots__ = ("reason", "reassess_in_seconds")
    REASON_FIELD_NUMBER: _ClassVar[int]
    REASSESS_IN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    reason: str
    reassess_in_seconds: int
    def __init__(self, reason: _Optional[str] = ..., reassess_in_seconds: _Optional[int] = ...) -> None: ...

class CostBenefit(_message.Message):
    __slots__ = ("cost_change_per_hour", "pods_that_will_schedule", "cluster_efficiency_gain", "estimated_waste_reduction", "disruption_risk", "spot_interruption_risk")
    COST_CHANGE_PER_HOUR_FIELD_NUMBER: _ClassVar[int]
    PODS_THAT_WILL_SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    CLUSTER_EFFICIENCY_GAIN_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_WASTE_REDUCTION_FIELD_NUMBER: _ClassVar[int]
    DISRUPTION_RISK_FIELD_NUMBER: _ClassVar[int]
    SPOT_INTERRUPTION_RISK_FIELD_NUMBER: _ClassVar[int]
    cost_change_per_hour: float
    pods_that_will_schedule: int
    cluster_efficiency_gain: float
    estimated_waste_reduction: float
    disruption_risk: float
    spot_interruption_risk: float
    def __init__(self, cost_change_per_hour: _Optional[float] = ..., pods_that_will_schedule: _Optional[int] = ..., cluster_efficiency_gain: _Optional[float] = ..., estimated_waste_reduction: _Optional[float] = ..., disruption_risk: _Optional[float] = ..., spot_interruption_risk: _Optional[float] = ...) -> None: ...

class MetricSnapshot(_message.Message):
    __slots__ = ("values", "ts")
    class ValuesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    VALUES_FIELD_NUMBER: _ClassVar[int]
    TS_FIELD_NUMBER: _ClassVar[int]
    values: _containers.ScalarMap[str, float]
    ts: _timestamp_pb2.Timestamp
    def __init__(self, values: _Optional[_Mapping[str, float]] = ..., ts: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ModelMetadata(_message.Message):
    __slots__ = ("model_version", "policy_name", "updated_at", "labels", "checkpoint_uri", "compatible_feature_schema")
    class LabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    POLICY_NAME_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_URI_FIELD_NUMBER: _ClassVar[int]
    COMPATIBLE_FEATURE_SCHEMA_FIELD_NUMBER: _ClassVar[int]
    model_version: str
    policy_name: str
    updated_at: _timestamp_pb2.Timestamp
    labels: _containers.ScalarMap[str, str]
    checkpoint_uri: str
    compatible_feature_schema: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, model_version: _Optional[str] = ..., policy_name: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., labels: _Optional[_Mapping[str, str]] = ..., checkpoint_uri: _Optional[str] = ..., compatible_feature_schema: _Optional[_Iterable[str]] = ...) -> None: ...

class RecommendationAppRequest(_message.Message):
    __slots__ = ("app", "dry_run", "snapshot")
    APP_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    app: AppRef
    dry_run: bool
    snapshot: MetricSnapshot
    def __init__(self, app: _Optional[_Union[AppRef, _Mapping]] = ..., dry_run: bool = ..., snapshot: _Optional[_Union[MetricSnapshot, _Mapping]] = ...) -> None: ...

class RecommendationAppResponse(_message.Message):
    __slots__ = ("plan", "decision_id", "model_version", "confidence", "audit_reasons", "effective_policy")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    AUDIT_REASONS_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_POLICY_FIELD_NUMBER: _ClassVar[int]
    plan: _containers.RepeatedCompositeFieldContainer[AppActionPlan]
    decision_id: str
    model_version: str
    confidence: float
    audit_reasons: _containers.RepeatedScalarFieldContainer[str]
    effective_policy: SafetyPolicy
    def __init__(self, plan: _Optional[_Iterable[_Union[AppActionPlan, _Mapping]]] = ..., decision_id: _Optional[str] = ..., model_version: _Optional[str] = ..., confidence: _Optional[float] = ..., audit_reasons: _Optional[_Iterable[str]] = ..., effective_policy: _Optional[_Union[SafetyPolicy, _Mapping]] = ...) -> None: ...

class ExecutionAppOutcome(_message.Message):
    __slots__ = ("decision_id", "app", "success", "note", "post_action_metrics", "reported_at")
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    APP_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    POST_ACTION_METRICS_FIELD_NUMBER: _ClassVar[int]
    REPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    decision_id: str
    app: AppRef
    success: bool
    note: str
    post_action_metrics: MetricSnapshot
    reported_at: _timestamp_pb2.Timestamp
    def __init__(self, decision_id: _Optional[str] = ..., app: _Optional[_Union[AppRef, _Mapping]] = ..., success: bool = ..., note: _Optional[str] = ..., post_action_metrics: _Optional[_Union[MetricSnapshot, _Mapping]] = ..., reported_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RecommendationClusterRequest(_message.Message):
    __slots__ = ("cluster", "dry_run", "analysis_window_hours", "snapshot")
    CLUSTER_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    ANALYSIS_WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    cluster: ClusterRef
    dry_run: bool
    analysis_window_hours: int
    snapshot: MetricSnapshot
    def __init__(self, cluster: _Optional[_Union[ClusterRef, _Mapping]] = ..., dry_run: bool = ..., analysis_window_hours: _Optional[int] = ..., snapshot: _Optional[_Union[MetricSnapshot, _Mapping]] = ...) -> None: ...

class RecommendationClusterResponse(_message.Message):
    __slots__ = ("plan", "decision_id", "model_version", "confidence", "audit_reasons")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    AUDIT_REASONS_FIELD_NUMBER: _ClassVar[int]
    plan: ClusterActionPlan
    decision_id: str
    model_version: str
    confidence: float
    audit_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, plan: _Optional[_Union[ClusterActionPlan, _Mapping]] = ..., decision_id: _Optional[str] = ..., model_version: _Optional[str] = ..., confidence: _Optional[float] = ..., audit_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class ExecutionClusterOutcome(_message.Message):
    __slots__ = ("decision_id", "cluster", "success", "note", "post_action_metrics", "reported_at")
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    CLUSTER_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    POST_ACTION_METRICS_FIELD_NUMBER: _ClassVar[int]
    REPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    decision_id: str
    cluster: ClusterRef
    success: bool
    note: str
    post_action_metrics: MetricSnapshot
    reported_at: _timestamp_pb2.Timestamp
    def __init__(self, decision_id: _Optional[str] = ..., cluster: _Optional[_Union[ClusterRef, _Mapping]] = ..., success: bool = ..., note: _Optional[str] = ..., post_action_metrics: _Optional[_Union[MetricSnapshot, _Mapping]] = ..., reported_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetAppActionRequest(_message.Message):
    __slots__ = ("app", "features", "candidates", "policy_override")
    class FeaturesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    APP_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    POLICY_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    app: AppRef
    features: _containers.ScalarMap[str, float]
    candidates: _containers.RepeatedCompositeFieldContainer[CandidateProposal]
    policy_override: SafetyPolicy
    def __init__(self, app: _Optional[_Union[AppRef, _Mapping]] = ..., features: _Optional[_Mapping[str, float]] = ..., candidates: _Optional[_Iterable[_Union[CandidateProposal, _Mapping]]] = ..., policy_override: _Optional[_Union[SafetyPolicy, _Mapping]] = ...) -> None: ...

class GetAppActionResponse(_message.Message):
    __slots__ = ("plan", "model_version", "confidence", "decision_id", "audit_reasons")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    AUDIT_REASONS_FIELD_NUMBER: _ClassVar[int]
    plan: AppActionPlan
    model_version: str
    confidence: float
    decision_id: str
    audit_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, plan: _Optional[_Union[AppActionPlan, _Mapping]] = ..., model_version: _Optional[str] = ..., confidence: _Optional[float] = ..., decision_id: _Optional[str] = ..., audit_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class EnsureModelResponse(_message.Message):
    __slots__ = ("model_version", "created", "meta")
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    META_FIELD_NUMBER: _ClassVar[int]
    model_version: str
    created: bool
    meta: ModelMetadata
    def __init__(self, model_version: _Optional[str] = ..., created: bool = ..., meta: _Optional[_Union[ModelMetadata, _Mapping]] = ...) -> None: ...

class TrainRequest(_message.Message):
    __slots__ = ("app", "reason", "horizon_hours", "base_version", "hparams")
    class HparamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    APP_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    HORIZON_HOURS_FIELD_NUMBER: _ClassVar[int]
    BASE_VERSION_FIELD_NUMBER: _ClassVar[int]
    HPARAMS_FIELD_NUMBER: _ClassVar[int]
    app: AppRef
    reason: str
    horizon_hours: int
    base_version: str
    hparams: _containers.ScalarMap[str, str]
    def __init__(self, app: _Optional[_Union[AppRef, _Mapping]] = ..., reason: _Optional[str] = ..., horizon_hours: _Optional[int] = ..., base_version: _Optional[str] = ..., hparams: _Optional[_Mapping[str, str]] = ...) -> None: ...

class TrainResponse(_message.Message):
    __slots__ = ("training_id", "job_name")
    TRAINING_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_NAME_FIELD_NUMBER: _ClassVar[int]
    training_id: str
    job_name: str
    def __init__(self, training_id: _Optional[str] = ..., job_name: _Optional[str] = ...) -> None: ...

class ListModelsRequest(_message.Message):
    __slots__ = ("app",)
    APP_FIELD_NUMBER: _ClassVar[int]
    app: AppRef
    def __init__(self, app: _Optional[_Union[AppRef, _Mapping]] = ...) -> None: ...

class ListModelsResponse(_message.Message):
    __slots__ = ("models",)
    MODELS_FIELD_NUMBER: _ClassVar[int]
    models: _containers.RepeatedCompositeFieldContainer[ModelMetadata]
    def __init__(self, models: _Optional[_Iterable[_Union[ModelMetadata, _Mapping]]] = ...) -> None: ...

class GetModelMetadataRequest(_message.Message):
    __slots__ = ("app", "model_version")
    APP_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    app: AppRef
    model_version: str
    def __init__(self, app: _Optional[_Union[AppRef, _Mapping]] = ..., model_version: _Optional[str] = ...) -> None: ...

class AgentRegistration(_message.Message):
    __slots__ = ("training_id", "agent_id", "version", "labels")
    class LabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TRAINING_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    training_id: str
    agent_id: str
    version: str
    labels: _containers.ScalarMap[str, str]
    def __init__(self, training_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., version: _Optional[str] = ..., labels: _Optional[_Mapping[str, str]] = ...) -> None: ...

class AgentRegistrationAck(_message.Message):
    __slots__ = ("accepted", "message")
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    message: str
    def __init__(self, accepted: bool = ..., message: _Optional[str] = ...) -> None: ...

class TrainingPollRequest(_message.Message):
    __slots__ = ("training_id", "agent_id")
    TRAINING_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    training_id: str
    agent_id: str
    def __init__(self, training_id: _Optional[str] = ..., agent_id: _Optional[str] = ...) -> None: ...

class TrainingSpec(_message.Message):
    __slots__ = ("training_id", "app", "horizon_hours", "base_version", "hparams", "output_uri", "clickhouse_dsn", "start_at", "end_at")
    class HparamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TRAINING_ID_FIELD_NUMBER: _ClassVar[int]
    APP_FIELD_NUMBER: _ClassVar[int]
    HORIZON_HOURS_FIELD_NUMBER: _ClassVar[int]
    BASE_VERSION_FIELD_NUMBER: _ClassVar[int]
    HPARAMS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_URI_FIELD_NUMBER: _ClassVar[int]
    CLICKHOUSE_DSN_FIELD_NUMBER: _ClassVar[int]
    START_AT_FIELD_NUMBER: _ClassVar[int]
    END_AT_FIELD_NUMBER: _ClassVar[int]
    training_id: str
    app: AppRef
    horizon_hours: int
    base_version: str
    hparams: _containers.ScalarMap[str, str]
    output_uri: str
    clickhouse_dsn: str
    start_at: _timestamp_pb2.Timestamp
    end_at: _timestamp_pb2.Timestamp
    def __init__(self, training_id: _Optional[str] = ..., app: _Optional[_Union[AppRef, _Mapping]] = ..., horizon_hours: _Optional[int] = ..., base_version: _Optional[str] = ..., hparams: _Optional[_Mapping[str, str]] = ..., output_uri: _Optional[str] = ..., clickhouse_dsn: _Optional[str] = ..., start_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TrainingProgress(_message.Message):
    __slots__ = ("training_id", "agent_id", "step", "train_loss", "eval_reward", "progress_pct", "ts", "scalars")
    class ScalarsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    TRAINING_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    TRAIN_LOSS_FIELD_NUMBER: _ClassVar[int]
    EVAL_REWARD_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_PCT_FIELD_NUMBER: _ClassVar[int]
    TS_FIELD_NUMBER: _ClassVar[int]
    SCALARS_FIELD_NUMBER: _ClassVar[int]
    training_id: str
    agent_id: str
    step: int
    train_loss: float
    eval_reward: float
    progress_pct: float
    ts: _timestamp_pb2.Timestamp
    scalars: _containers.ScalarMap[str, float]
    def __init__(self, training_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., step: _Optional[int] = ..., train_loss: _Optional[float] = ..., eval_reward: _Optional[float] = ..., progress_pct: _Optional[float] = ..., ts: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., scalars: _Optional[_Mapping[str, float]] = ...) -> None: ...

class TrainingResult(_message.Message):
    __slots__ = ("training_id", "agent_id", "success", "error_message", "model_version", "checkpoint_uri", "eval_reward", "metrics", "ts")
    class MetricsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    TRAINING_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_URI_FIELD_NUMBER: _ClassVar[int]
    EVAL_REWARD_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    TS_FIELD_NUMBER: _ClassVar[int]
    training_id: str
    agent_id: str
    success: bool
    error_message: str
    model_version: str
    checkpoint_uri: str
    eval_reward: float
    metrics: _containers.ScalarMap[str, float]
    ts: _timestamp_pb2.Timestamp
    def __init__(self, training_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., success: bool = ..., error_message: _Optional[str] = ..., model_version: _Optional[str] = ..., checkpoint_uri: _Optional[str] = ..., eval_reward: _Optional[float] = ..., metrics: _Optional[_Mapping[str, float]] = ..., ts: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AgentHeartbeat(_message.Message):
    __slots__ = ("training_id", "agent_id", "ts", "sysinfo")
    class SysinfoEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TRAINING_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    TS_FIELD_NUMBER: _ClassVar[int]
    SYSINFO_FIELD_NUMBER: _ClassVar[int]
    training_id: str
    agent_id: str
    ts: _timestamp_pb2.Timestamp
    sysinfo: _containers.ScalarMap[str, str]
    def __init__(self, training_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., ts: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., sysinfo: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CancelTrainingRequest(_message.Message):
    __slots__ = ("training_id", "reason")
    TRAINING_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    training_id: str
    reason: str
    def __init__(self, training_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class CancelTrainingAck(_message.Message):
    __slots__ = ("acknowledged", "message")
    ACKNOWLEDGED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    acknowledged: bool
    message: str
    def __init__(self, acknowledged: bool = ..., message: _Optional[str] = ...) -> None: ...
