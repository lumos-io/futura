import { CloudProvider, cloudProviderFromJSON, ProviderConnection } from "@proto/backend/backend";
import {
    ClusterMetadata
} from "@proto/backend/cluster";

type Cluster = ClusterMetadata;

export function GetClusterName(cluster: Cluster, provider: ProviderConnection): string | undefined {
    switch (cloudProviderFromJSON(provider.provider)) {
        case CloudProvider.ALIBABA:
            return cluster.ack_metadata?.name;
        case CloudProvider.AZURE:
            return cluster.aks_metadata?.name;
        case CloudProvider.AWS:
            return cluster.eks_metadata?.name;
        case CloudProvider.DIGITALOCEAN:
            return cluster.doks_metadata?.name;
        case CloudProvider.GCP:
            return cluster.gke_metadata?.name;
        case CloudProvider.KIND:
            return cluster.kind_metadata?.name;
    }
    return undefined
}

export default Cluster;