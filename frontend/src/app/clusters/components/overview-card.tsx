import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatProtoOrDate } from "@/lib/utils";
import { CloudProviderConnection } from "@/models/cloud-provider";
import { Cluster } from "@/models/kubernetes";
import { CloudProvider, cloudProviderFromJSON } from "@proto/backend/backend";

type ClusterCardProps = {
  provider: CloudProviderConnection | undefined;
  cluster: Cluster;
};

function renderClusterCardContent({ provider, cluster }: ClusterCardProps) {
  switch (cloudProviderFromJSON(provider?.provider)) {
    case CloudProvider.AWS: {
      if (!cluster.eks_metadata) {
        return null;
      }
      const metadata = cluster.eks_metadata;
      return (
        <CardContent>
          <p className="text-sm font-medium">EKS Cluster</p>
          <p>Status: {metadata.status}</p>
          <p>Version: {metadata.version}</p>
          <p>Platform: {metadata.platform_version}</p>
          <p>Endpoint: {metadata.endpoint}</p>
        </CardContent>
      );
    }

    case CloudProvider.AZURE: {
      if (!cluster.aks_metadata) {
        return null;
      }
      const metadata = cluster.aks_metadata;
      return (
        <CardContent>
          <p className="text-sm font-medium">EKS Cluster</p>
          <p>Id: {metadata.cluster_metadata_id}</p>
        </CardContent>
      );
    }

    case CloudProvider.GCP: {
      if (!cluster.gke_metadata) {
        return null;
      }
      const metadata = cluster.gke_metadata;
      return (
        <CardContent>
          <p className="text-sm font-medium">EKS Cluster</p>
          <p>Id: {metadata.cluster_metadata_id}</p>
        </CardContent>
      );
    }

    case CloudProvider.DIGITALOCEAN: {
      if (!cluster.doks_metadata) {
        return null;
      }
      const metadata = cluster.doks_metadata;
      return (
        <CardContent>
          <p className="text-sm font-medium">EKS Cluster</p>
          <p>Id: {metadata.cluster_metadata_id}</p>
        </CardContent>
      );
    }

    case CloudProvider.ALIBABA: {
      if (!cluster.ack_metadata) {
        return null;
      }
      const metadata = cluster.ack_metadata;
      return (
        <CardContent>
          <p className="text-sm font-medium">EKS Cluster</p>
          <p>Id: {metadata.cluster_metadata_id}</p>
        </CardContent>
      );
    }

    case CloudProvider.KIND: {
      if (!cluster.kind_metadata) {
        return null;
      }
      const metadata = cluster.kind_metadata;
      return (
        <CardContent>
          <p className="text-sm font-medium">Kind Cluster (Dev)</p>
          <p>Status: {metadata.status}</p>
          <p>Version: {metadata.version}</p>
          <p>Platform: {metadata.platform_version}</p>
          <p>Endpoint: {metadata.endpoint}</p>
          <p>
            Created At:{" "}
            {metadata.cluster_created_at
              ? formatProtoOrDate(metadata.cluster_created_at)
              : ""}
          </p>
          {metadata.tags &&
            Object.entries(metadata.tags).map(([key, value]) => (
              <Badge variant="secondary">
                {key}:{value}
              </Badge>
            ))}
        </CardContent>
      );
    }

    default:
      return (
        <CardContent>
          <p className="text-sm font-medium">Unknown Cluster Type</p>
        </CardContent>
      );
  }
}

const OverviewClusterCard: React.FC<{
  provider: CloudProviderConnection | undefined;
  cluster: Cluster;
}> = (props) => {
  return (
    <Card key={props.cluster.id}>
      <CardHeader>
        <CardTitle>{props.provider ? props.cluster.name : ""}</CardTitle>
        <CardDescription>Api Key: {props.cluster.api_key}</CardDescription>
      </CardHeader>
      {renderClusterCardContent({
        provider: props.provider,
        cluster: props.cluster,
      })}
    </Card>
  );
};

export default OverviewClusterCard;
