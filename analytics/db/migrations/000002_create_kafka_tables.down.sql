DROP TABLE IF EXISTS kubernetes_events_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubernetes_events
-- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
DROP TABLE IF EXISTS kubernetes_objects_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubernetes_objects
DROP TABLE IF EXISTS kubernetes_containers_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubernetes_containers
DROP TABLE IF EXISTS kubernetes_volumes_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubernetes_volumes
DROP TABLE IF EXISTS kubernetes_node_conditions_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubernetes_node_conditions
DROP TABLE IF EXISTS kubernetes_allocatable_resources_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubernetes_allocatable_resources
DROP TABLE IF EXISTS kubernetes_cluster_quotas_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubernetes_cluster_quotas
DROP TABLE IF EXISTS kubernetes_namespace_quotas_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubernetes_namespace_quotas
-- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
DROP TABLE IF EXISTS kubelet_node_metrics_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubelet_node_metrics
DROP TABLE IF EXISTS kubelet_pod_metrics_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubelet_pod_metrics
DROP TABLE IF EXISTS kubelet_container_metrics_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubelet_container_metrics
DROP TABLE IF EXISTS kubelet_network_metrics_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubelet_network_metrics
DROP TABLE IF EXISTS kubelet_volume_metrics_kafka
DROP MATERIALIZED VIEW IF EXISTS mv_kubelet_volume_metrics