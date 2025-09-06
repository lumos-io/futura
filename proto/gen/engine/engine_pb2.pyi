import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ClusterOptimizationConfig(_message.Message):
    __slots__ = ("api_key", "cloud_provider", "region", "cost_sensitivity", "monthly_budget", "preferred_instance_types", "allow_spot", "max_spot_percentage")
    API_KEY_FIELD_NUMBER: _ClassVar[int]
    CLOUD_PROVIDER_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    COST_SENSITIVITY_FIELD_NUMBER: _ClassVar[int]
    MONTHLY_BUDGET_FIELD_NUMBER: _ClassVar[int]
    PREFERRED_INSTANCE_TYPES_FIELD_NUMBER: _ClassVar[int]
    ALLOW_SPOT_FIELD_NUMBER: _ClassVar[int]
    MAX_SPOT_PERCENTAGE_FIELD_NUMBER: _ClassVar[int]
    api_key: str
    cloud_provider: str
    region: str
    cost_sensitivity: str
    monthly_budget: str
    preferred_instance_types: _containers.RepeatedScalarFieldContainer[str]
    allow_spot: bool
    max_spot_percentage: str
    def __init__(self, api_key: _Optional[str] = ..., cloud_provider: _Optional[str] = ..., region: _Optional[str] = ..., cost_sensitivity: _Optional[str] = ..., monthly_budget: _Optional[str] = ..., preferred_instance_types: _Optional[_Iterable[str]] = ..., allow_spot: bool = ..., max_spot_percentage: _Optional[str] = ...) -> None: ...

class ClusterOptimizationConfigRequest(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: ClusterOptimizationConfig
    def __init__(self, config: _Optional[_Union[ClusterOptimizationConfig, _Mapping]] = ...) -> None: ...

class ClusterOptimizationConfigResponse(_message.Message):
    __slots__ = ("success", "message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    def __init__(self, success: bool = ..., message: _Optional[str] = ...) -> None: ...

class ServiceLevelObjective(_message.Message):
    __slots__ = ("service_name", "target_p95_latency", "target_error_rate", "target_throughput", "priority", "last_updated")
    SERVICE_NAME_FIELD_NUMBER: _ClassVar[int]
    TARGET_P95_LATENCY_FIELD_NUMBER: _ClassVar[int]
    TARGET_ERROR_RATE_FIELD_NUMBER: _ClassVar[int]
    TARGET_THROUGHPUT_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATED_FIELD_NUMBER: _ClassVar[int]
    service_name: str
    target_p95_latency: str
    target_error_rate: str
    target_throughput: str
    priority: str
    last_updated: _timestamp_pb2.Timestamp
    def __init__(self, service_name: _Optional[str] = ..., target_p95_latency: _Optional[str] = ..., target_error_rate: _Optional[str] = ..., target_throughput: _Optional[str] = ..., priority: _Optional[str] = ..., last_updated: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SyncSLORequest(_message.Message):
    __slots__ = ("slo", "api_key")
    SLO_FIELD_NUMBER: _ClassVar[int]
    API_KEY_FIELD_NUMBER: _ClassVar[int]
    slo: ServiceLevelObjective
    api_key: str
    def __init__(self, slo: _Optional[_Union[ServiceLevelObjective, _Mapping]] = ..., api_key: _Optional[str] = ...) -> None: ...

class SyncSLOResponse(_message.Message):
    __slots__ = ("success", "message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    def __init__(self, success: bool = ..., message: _Optional[str] = ...) -> None: ...

class DecisionRequest(_message.Message):
    __slots__ = ("cluster_id",)
    CLUSTER_ID_FIELD_NUMBER: _ClassVar[int]
    cluster_id: str
    def __init__(self, cluster_id: _Optional[str] = ...) -> None: ...

class TargetRef(_message.Message):
    __slots__ = ("kind", "namespace", "name")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    kind: str
    namespace: str
    name: str
    def __init__(self, kind: _Optional[str] = ..., namespace: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class HpaScaleAction(_message.Message):
    __slots__ = ("replicas",)
    REPLICAS_FIELD_NUMBER: _ClassVar[int]
    replicas: int
    def __init__(self, replicas: _Optional[int] = ...) -> None: ...

class KarpenterProvisionAction(_message.Message):
    __slots__ = ("instance_types", "count", "capacity_type")
    INSTANCE_TYPES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_TYPE_FIELD_NUMBER: _ClassVar[int]
    instance_types: _containers.RepeatedScalarFieldContainer[str]
    count: int
    capacity_type: str
    def __init__(self, instance_types: _Optional[_Iterable[str]] = ..., count: _Optional[int] = ..., capacity_type: _Optional[str] = ...) -> None: ...

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

class Action(_message.Message):
    __slots__ = ("type", "confidence", "reason", "hpa_scale", "karpenter_provision", "vpa_recommend")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    HPA_SCALE_FIELD_NUMBER: _ClassVar[int]
    KARPENTER_PROVISION_FIELD_NUMBER: _ClassVar[int]
    VPA_RECOMMEND_FIELD_NUMBER: _ClassVar[int]
    type: str
    confidence: float
    reason: str
    hpa_scale: HpaScaleAction
    karpenter_provision: KarpenterProvisionAction
    vpa_recommend: VpaRecommendAction
    def __init__(self, type: _Optional[str] = ..., confidence: _Optional[float] = ..., reason: _Optional[str] = ..., hpa_scale: _Optional[_Union[HpaScaleAction, _Mapping]] = ..., karpenter_provision: _Optional[_Union[KarpenterProvisionAction, _Mapping]] = ..., vpa_recommend: _Optional[_Union[VpaRecommendAction, _Mapping]] = ...) -> None: ...

class ExpectedOutcomes(_message.Message):
    __slots__ = ("predicted_latency_p95_ms", "predicted_cost_delta_per_hour_usd")
    PREDICTED_LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    PREDICTED_COST_DELTA_PER_HOUR_USD_FIELD_NUMBER: _ClassVar[int]
    predicted_latency_p95_ms: float
    predicted_cost_delta_per_hour_usd: float
    def __init__(self, predicted_latency_p95_ms: _Optional[float] = ..., predicted_cost_delta_per_hour_usd: _Optional[float] = ...) -> None: ...

class DecisionResponse(_message.Message):
    __slots__ = ("decision_id", "target", "actions", "expected_outcomes")
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    decision_id: str
    target: TargetRef
    actions: _containers.RepeatedCompositeFieldContainer[Action]
    expected_outcomes: ExpectedOutcomes
    def __init__(self, decision_id: _Optional[str] = ..., target: _Optional[_Union[TargetRef, _Mapping]] = ..., actions: _Optional[_Iterable[_Union[Action, _Mapping]]] = ..., expected_outcomes: _Optional[_Union[ExpectedOutcomes, _Mapping]] = ...) -> None: ...
