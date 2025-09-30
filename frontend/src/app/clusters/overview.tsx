import React, { useEffect, useState } from "react";
import { Cluster } from "@/models/kubernetes";
import { useAuth } from "@/hooks/auth-provider";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { CloudProviderConnection } from "@/models/cloud-provider";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Activity,
  AlertCircle,
  Box,
  Boxes,
  Calendar,
  Cpu,
  DollarSign,
  HardDrive,
  Layers,
  Network,
  Server,
} from "lucide-react";

type ClusterMetrics = {
  nodes: {
    total: number;
    ready: number;
    notReady: number;
  };
  workloads: {
    deployments: number;
    statefulSets: number;
    daemonSets: number;
    jobs: number;
    cronJobs: number;
  };
  pods: {
    total: number;
    running: number;
    pending: number;
    failed: number;
  };
  resources: {
    cpuCapacity: string;
    cpuUsage: string;
    memoryCapacity: string;
    memoryUsage: string;
  };
  namespaces: number;
  services: number;
  events: {
    total: number;
    warnings: number;
    errors: number;
  };
  storage: {
    pvcs: number;
    totalCapacity: string;
  };
  cost: {
    estimated: string;
    currency: string;
  };
};

const ClustersOverview: React.FC = () => {
  const { user } = useAuth();

  const [providersConnection, setProvidersConnection] = useState<
    CloudProviderConnection[]
  >([]);
  const [selectedProvider, setSelectedProvider] =
    useState<CloudProviderConnection>();
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [selectedCluster, setSelectedCluster] = useState<Cluster>();
  const [metrics, setMetrics] = useState<ClusterMetrics | null>(null);
  const [loading, setLoading] = useState(false);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const orgId = user?.organizationId;

  useEffect(() => {
    const fetchProviders = async () => {
      if (!orgId) {
        return;
      }

      const res = await fetch(`/api/organizations/${orgId}/connects`);
      if (!res.ok) {
        return;
      }

      const data = await res.json();
      setProvidersConnection(data.data);
    };

    fetchProviders();
  }, [orgId]);

  useEffect(() => {
    if (providersConnection.length > 0 && !selectedProvider) {
      setSelectedProvider(providersConnection[0]);
    }
  }, [providersConnection, selectedProvider]);

  useEffect(() => {
    if (!selectedProvider) {
      return;
    }

    const fetchClusters = async () => {
      setLoading(true);
      setError(null);

      try {
        const res = await fetch(
          `/api/organizations/${orgId}/connects/${selectedProvider.id}/clusters`
        );
        if (!res.ok) {
          throw new Error("Failed to fetch clusters");
        }

        const data = await res.json();
        const fetchedClusters: Cluster[] = data.data ?? [];
        setClusters(fetchedClusters);
        setSelectedCluster(fetchedClusters[0]);
      } catch (err) {
        console.error(err);
        setError("Unknown error");
        setClusters([]);
        setSelectedCluster(undefined);
      } finally {
        setLoading(false);
      }
    };

    fetchClusters();
  }, [selectedProvider, orgId]);

  // Fetch cluster metrics when cluster is selected
  useEffect(() => {
    if (!selectedCluster || !orgId) {
      return;
    }

    const fetchMetrics = async () => {
      setMetricsLoading(true);
      try {
        const res = await fetch(
          `/api/organizations/${orgId}/clusters/${selectedCluster.id}/metrics`
        );
        if (!res.ok) {
          throw new Error("Failed to fetch metrics");
        }

        const data = await res.json();
        setMetrics(data.data);
      } catch (err) {
        console.error("Error fetching metrics:", err);
        // Set mock data for development
        setMetrics({
          nodes: { total: 3, ready: 3, notReady: 0 },
          workloads: {
            deployments: 12,
            statefulSets: 3,
            daemonSets: 5,
            jobs: 8,
            cronJobs: 2,
          },
          pods: { total: 45, running: 42, pending: 2, failed: 1 },
          resources: {
            cpuCapacity: "12 cores",
            cpuUsage: "8.4 cores (70%)",
            memoryCapacity: "48 GB",
            memoryUsage: "32 GB (67%)",
          },
          namespaces: 8,
          services: 15,
          events: { total: 124, warnings: 3, errors: 1 },
          storage: { pvcs: 10, totalCapacity: "500 GB" },
          cost: { estimated: "1,245.00", currency: "USD" },
        });
      } finally {
        setMetricsLoading(false);
      }
    };

    fetchMetrics();
  }, [selectedCluster, orgId]);

  const getClusterVersion = () => {
    if (!selectedCluster) return "N/A";
    if (selectedCluster.eks_metadata) return selectedCluster.eks_metadata.version;
    if (selectedCluster.kind_metadata) return selectedCluster.kind_metadata.version;
    return "N/A";
  };

  const getClusterStatus = () => {
    if (!selectedCluster) return "unknown";
    if (selectedCluster.eks_metadata) return selectedCluster.eks_metadata.status?.toLowerCase() || "unknown";
    if (selectedCluster.kind_metadata) return selectedCluster.kind_metadata.status?.toLowerCase() || "unknown";
    return "unknown";
  };

  const getStatusBadgeVariant = (status: string) => {
    if (status === "active" || status === "running") return "default";
    if (status === "creating") return "secondary";
    return "destructive";
  };

  return (
    <div className="p-10 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-semibold">Cluster Overview</h1>
          <p className="text-muted-foreground mt-1">
            Monitor your Kubernetes cluster health and resources
          </p>
        </div>
      </div>

      <div className="flex flex-col md:flex-row gap-4">
        <div className="w-full md:w-60">
          <Select
            value={selectedProvider?.id.toString() ?? ""}
            onValueChange={(value) => {
              const found = providersConnection.find(
                (p) => p.id.toString() === value
              );
              if (found) {
                setSelectedProvider(found);
              }
            }}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select Cloud Provider" />
            </SelectTrigger>
            <SelectContent>
              {providersConnection.map((provider) => (
                <SelectItem key={provider.id} value={provider.id.toString()}>
                  {provider.connection_name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="w-full md:w-60">
          <Select
            value={selectedCluster?.id.toString() ?? ""}
            onValueChange={(value) => {
              const found = clusters.find((c) => c.id.toString() === value);
              if (found) {
                setSelectedCluster(found);
              }
            }}
            disabled={clusters.length === 0}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select Cluster" />
            </SelectTrigger>
            <SelectContent>
              {clusters.map((cluster) => (
                <SelectItem key={cluster.id} value={cluster.id.toString()}>
                  {cluster.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {loading && <p className="text-muted-foreground">Loading clusters...</p>}
      {error && <p className="text-destructive">{error}</p>}
      {clusters.length === 0 && !loading && !error && (
        <p className="text-muted-foreground text-center py-8">
          No clusters found for{" "}
          {selectedProvider
            ? selectedProvider.connection_name
            : "this provider"}
          .
        </p>
      )}

      {selectedCluster && (
        <>
          {/* Cluster Info Card */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-2xl">{selectedCluster.name}</CardTitle>
                  <CardDescription className="mt-1">
                    API Key: {selectedCluster.api_key}
                  </CardDescription>
                </div>
                <Badge variant={getStatusBadgeVariant(getClusterStatus())}>
                  {getClusterStatus()}
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="flex items-center gap-2">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <div>
                    <p className="text-xs text-muted-foreground">Version</p>
                    <p className="font-medium">{getClusterVersion()}</p>
                  </div>
                </div>
                {selectedCluster.eks_metadata?.endpoint && (
                  <div className="flex items-center gap-2 col-span-2">
                    <Network className="h-4 w-4 text-muted-foreground" />
                    <div>
                      <p className="text-xs text-muted-foreground">Endpoint</p>
                      <p className="font-medium text-xs truncate">
                        {selectedCluster.eks_metadata.endpoint}
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {metricsLoading ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              {[...Array(8)].map((_, i) => (
                <Card key={i}>
                  <CardHeader>
                    <Skeleton className="h-4 w-24" />
                  </CardHeader>
                  <CardContent>
                    <Skeleton className="h-8 w-16" />
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : metrics ? (
            <>
              {/* Key Metrics */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-sm font-medium">Nodes</CardTitle>
                    <Server className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{metrics.nodes.total}</div>
                    <p className="text-xs text-muted-foreground">
                      {metrics.nodes.ready} ready, {metrics.nodes.notReady} not ready
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-sm font-medium">Pods</CardTitle>
                    <Box className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{metrics.pods.total}</div>
                    <p className="text-xs text-muted-foreground">
                      {metrics.pods.running} running, {metrics.pods.pending} pending
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-sm font-medium">Namespaces</CardTitle>
                    <Layers className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{metrics.namespaces}</div>
                    <p className="text-xs text-muted-foreground">
                      Active namespaces
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-sm font-medium">Est. Cost</CardTitle>
                    <DollarSign className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">
                      ${metrics.cost.estimated}
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {metrics.cost.currency} / month
                    </p>
                  </CardContent>
                </Card>
              </div>

              {/* Workloads & Resources */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Boxes className="h-5 w-5" />
                      Workloads
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-sm">Deployments</span>
                      <Badge variant="secondary">{metrics.workloads.deployments}</Badge>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-sm">StatefulSets</span>
                      <Badge variant="secondary">{metrics.workloads.statefulSets}</Badge>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-sm">DaemonSets</span>
                      <Badge variant="secondary">{metrics.workloads.daemonSets}</Badge>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-sm">Jobs</span>
                      <Badge variant="secondary">{metrics.workloads.jobs}</Badge>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-sm">CronJobs</span>
                      <Badge variant="secondary">{metrics.workloads.cronJobs}</Badge>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Cpu className="h-5 w-5" />
                      Resources
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div>
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-sm">CPU</span>
                        <span className="text-sm font-medium">{metrics.resources.cpuUsage}</span>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        Capacity: {metrics.resources.cpuCapacity}
                      </p>
                    </div>
                    <div>
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-sm">Memory</span>
                        <span className="text-sm font-medium">{metrics.resources.memoryUsage}</span>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        Capacity: {metrics.resources.memoryCapacity}
                      </p>
                    </div>
                  </CardContent>
                </Card>
              </div>

              {/* Additional Metrics */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-sm font-medium">Services</CardTitle>
                    <Activity className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{metrics.services}</div>
                    <p className="text-xs text-muted-foreground">
                      Active services
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-sm font-medium">Events</CardTitle>
                    <AlertCircle className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{metrics.events.total}</div>
                    <p className="text-xs text-muted-foreground">
                      {metrics.events.warnings} warnings, {metrics.events.errors} errors
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-sm font-medium">Storage</CardTitle>
                    <HardDrive className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{metrics.storage.pvcs}</div>
                    <p className="text-xs text-muted-foreground">
                      PVCs, {metrics.storage.totalCapacity} total
                    </p>
                  </CardContent>
                </Card>
              </div>
            </>
          ) : null}
        </>
      )}
    </div>
  );
};

export default ClustersOverview;
