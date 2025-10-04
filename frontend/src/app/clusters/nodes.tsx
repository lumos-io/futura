import React, { useEffect, useState } from "react";
import { useAuth } from "@/hooks/auth-provider";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Progress } from "@/components/ui/progress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Server,
  AlertTriangle,
  CheckCircle2,
  Settings2,
  DollarSign,
  Zap,
} from "lucide-react";
import { Cluster } from "@/models/kubernetes";
import { CloudProviderConnection } from "@/models/cloud-provider";

type Node = {
  name: string;
  status: "Ready" | "NotReady" | "Unknown";
  role: string;
  age: string;
  version: string;
  instanceType: string;
  zone: string;
  cpu: {
    capacity: string;
    usage: string;
    usagePercent: number;
  };
  memory: {
    capacity: string;
    usage: string;
    usagePercent: number;
  };
  pods: {
    capacity: number;
    current: number;
  };
  capacityType: "on-demand" | "spot";
  costPerHour: number;
};

type ClusterConfig = {
  clusterScalingMode: "auto" | "recommend";
  enableHPA: boolean;
  enableVPA: boolean;
  enableNodeHandler: boolean;
  costSensitivity: string;
  monthlyBudget: string;
  spotInstancesAllowed: boolean;
  maxSpotPercentage: string;
};

type ScalingRecommendation = {
  type: "provision" | "deprovision" | "no_action";
  confidence: number;
  urgency: "low" | "medium" | "high";
  reason: string;
  estimatedCostImpact: string;
  nodeGroups?: {
    name: string;
    count: number;
    instanceType: string[];
    reason: string;
  }[];
  nodesToRemove?: string[];
};

interface NodesProps {
  title: string;
}

const ClusterNodes: React.FC<NodesProps> = ({ title }) => {
  const { user } = useAuth();
  const [providersConnection, setProvidersConnection] = useState<
    CloudProviderConnection[]
  >([]);
  const [selectedProvider, setSelectedProvider] =
    useState<CloudProviderConnection>();
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [selectedCluster, setSelectedCluster] = useState<Cluster>();
  const [nodes, setNodes] = useState<Node[]>([]);
  const [clusterConfig, setClusterConfig] = useState<ClusterConfig | null>(
    null
  );
  const [recommendation] = useState<ScalingRecommendation | null>(null);
  const [loading, setLoading] = useState(false);

  const orgId = user?.organizationId;

  useEffect(() => {
    const fetchProviders = async () => {
      if (!orgId) return;

      const res = await fetch(`/api/organizations/${orgId}/connects`);
      if (res.ok) {
        const data = await res.json();
        setProvidersConnection(data.data);
      }
    };

    fetchProviders();
  }, [orgId]);

  useEffect(() => {
    if (providersConnection.length > 0 && !selectedProvider) {
      setSelectedProvider(providersConnection[0]);
    }
  }, [providersConnection, selectedProvider]);

  useEffect(() => {
    const fetchClusters = async () => {
      if (!selectedProvider || !orgId) return;

      setLoading(true);
      const res = await fetch(
        `/api/organizations/${orgId}/connects/${selectedProvider.id}/clusters`
      );

      if (res.ok) {
        const data = await res.json();
        setClusters(data.data);
        if (data.data.length > 0) {
          setSelectedCluster(data.data[0]);
        }
      }
      setLoading(false);
    };

    fetchClusters();
  }, [selectedProvider, orgId]);

  useEffect(() => {
    if (!selectedCluster || !orgId) return;

    setLoading(true);

    // Connect to SSE endpoint for real-time node data
    const eventSource = new EventSource(
      `/api/organizations/${orgId}/clusters/${selectedCluster.id}/nodes/stream`
    );

    eventSource.addEventListener("nodes-data", (event) => {
      try {
        const data = JSON.parse(event.data);

        // Process nodes data
        if (data.nodes) {
          const processedNodes: Node[] = data.nodes.map((node: { name: string; status: string; kubeletVersion?: string; instanceType?: string; zone?: string }) => ({
            name: node.name,
            status: node.status as "Ready" | "NotReady" | "Unknown",
            role: "worker", // TODO: Extract from labels
            age: "N/A", // TODO: Calculate from timestamp
            version: node.kubeletVersion || "N/A",
            instanceType: node.instanceType || "N/A",
            zone: node.zone || "N/A",
            cpu: {
              capacity: node.allocatable?.cpu || "0",
              usage: "N/A", // TODO: Get from metrics
              usagePercent: 0, // TODO: Calculate from metrics
            },
            memory: {
              capacity: node.allocatable?.memory || "0",
              usage: "N/A", // TODO: Get from metrics
              usagePercent: 0, // TODO: Calculate from metrics
            },
            pods: {
              capacity: parseInt(node.allocatable?.pods || "0"),
              current: node.podCount || 0,
            },
            capacityType: node.capacityType as "on-demand" | "spot",
            costPerHour: 0, // TODO: Calculate from instance type
          }));

          setNodes(processedNodes);
        }

        // Process cluster config
        if (data.config) {
          const config: ClusterConfig = {
            clusterScalingMode: data.config.clusterScalingMode || "recommend",
            enableHPA: data.config.enableHpa || false,
            enableVPA: data.config.enableVpa || false,
            enableNodeHandler: data.config.enableNodeHandler || false,
            costSensitivity: data.config.costSensitivity || "balanced",
            monthlyBudget: data.config.monthlyBudget || "$0",
            spotInstancesAllowed: data.config.spotInstancesAllowed || false,
            maxSpotPercentage: data.config.maxSpotPercentage || "0%",
          };

          setClusterConfig(config);
        }

        setLoading(false);
      } catch (error) {
        console.error("Failed to parse SSE data:", error);
      }
    });

    eventSource.onerror = (error) => {
      console.error("SSE error:", error);
      eventSource.close();
      setLoading(false);
    };

    // Cleanup on unmount or cluster change
    return () => {
      eventSource.close();
    };
  }, [selectedCluster, orgId]);

  const getUtilizationColor = (percent: number) => {
    if (percent >= 80) return "text-red-600";
    if (percent >= 60) return "text-yellow-600";
    return "text-green-600";
  };

  const getUrgencyColor = (urgency: string) => {
    if (urgency === "high") return "destructive";
    if (urgency === "medium") return "default";
    return "secondary";
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold">{title}</h1>
      </div>

      {/* Provider and Cluster Selection */}
      <div className="flex gap-4">
        <Select
          value={selectedProvider?.id.toString() ?? ""}
          onValueChange={(value) => {
            const provider = providersConnection.find(
              (p) => p.id.toString() === value
            );
            if (provider) {
              setSelectedProvider(provider);
            }
          }}
        >
          <SelectTrigger className="w-[250px]">
            <SelectValue placeholder="Select provider" />
          </SelectTrigger>
          <SelectContent>
            {providersConnection.map((provider) => (
              <SelectItem key={provider.id} value={provider.id.toString()}>
                {provider.connection_name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={selectedCluster?.id.toString() ?? ""}
          onValueChange={(value) => {
            const cluster = clusters.find((c) => c.id.toString() === value);
            if (cluster) {
              setSelectedCluster(cluster);
            }
          }}
        >
          <SelectTrigger className="w-[250px]">
            <SelectValue placeholder="Select cluster" />
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

      {loading ? (
        <div className="space-y-4">
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      ) : (
        <>
          {/* Cluster Configuration */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Settings2 className="h-5 w-5" />
                Cluster Optimization Configuration
              </CardTitle>
              <CardDescription>
                Current optimization settings for {selectedCluster?.name}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">Scaling Mode</p>
                  <Badge
                    variant={
                      clusterConfig?.clusterScalingMode === "auto"
                        ? "default"
                        : "secondary"
                    }
                    className="mt-1"
                  >
                    {clusterConfig?.clusterScalingMode === "auto"
                      ? "Automatic"
                      : "Recommend Only"}
                  </Badge>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">
                    Cost Sensitivity
                  </p>
                  <p className="font-medium capitalize">
                    {clusterConfig?.costSensitivity}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">
                    Monthly Budget
                  </p>
                  <p className="font-medium">{clusterConfig?.monthlyBudget}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">
                    Spot Instances
                  </p>
                  <p className="font-medium">
                    {clusterConfig?.spotInstancesAllowed
                      ? `Allowed (${clusterConfig.maxSpotPercentage})`
                      : "Not Allowed"}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">HPA</p>
                  <Badge
                    variant={clusterConfig?.enableHPA ? "default" : "outline"}
                  >
                    {clusterConfig?.enableHPA ? "Enabled" : "Disabled"}
                  </Badge>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">VPA</p>
                  <Badge
                    variant={clusterConfig?.enableVPA ? "default" : "outline"}
                  >
                    {clusterConfig?.enableVPA ? "Enabled" : "Disabled"}
                  </Badge>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Node Handler</p>
                  <Badge
                    variant={
                      clusterConfig?.enableNodeHandler ? "default" : "outline"
                    }
                  >
                    {clusterConfig?.enableNodeHandler ? "Enabled" : "Disabled"}
                  </Badge>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Scaling Recommendations */}
          {recommendation &&
            clusterConfig?.clusterScalingMode === "recommend" && (
              <Card className="border-l-4 border-l-blue-500">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Zap className="h-5 w-5 text-blue-500" />
                    Scaling Recommendation
                  </CardTitle>
                  <CardDescription>
                    AI-powered cluster scaling suggestion
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-start gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Badge
                          variant={getUrgencyColor(recommendation.urgency)}
                        >
                          {recommendation.urgency.toUpperCase()} URGENCY
                        </Badge>
                        <Badge variant="outline">
                          {(recommendation.confidence * 100).toFixed(0)}%
                          Confidence
                        </Badge>
                      </div>
                      <p className="text-sm text-muted-foreground mb-3">
                        {recommendation.reason}
                      </p>
                      <div className="flex items-center gap-2 text-sm">
                        <DollarSign className="h-4 w-4" />
                        <span className="font-medium">
                          Estimated Cost Impact:{" "}
                          <span
                            className={
                              recommendation.estimatedCostImpact.startsWith("+")
                                ? "text-red-600"
                                : "text-green-600"
                            }
                          >
                            {recommendation.estimatedCostImpact}
                          </span>
                        </span>
                      </div>
                    </div>
                  </div>

                  {recommendation.type === "provision" &&
                    recommendation.nodeGroups && (
                      <div className="space-y-2">
                        <h4 className="font-medium text-sm">
                          Recommended Node Groups:
                        </h4>
                        {recommendation.nodeGroups.map((group, idx) => (
                          <div
                            key={idx}
                            className="border rounded-lg p-3 space-y-1"
                          >
                            <div className="flex items-center justify-between">
                              <span className="font-medium">{group.name}</span>
                              <Badge variant="secondary">
                                {group.count} node{group.count > 1 ? "s" : ""}
                              </Badge>
                            </div>
                            <p className="text-xs text-muted-foreground">
                              Types: {group.instanceType.join(", ")}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {group.reason}
                            </p>
                          </div>
                        ))}
                      </div>
                    )}

                  {recommendation.type === "deprovision" &&
                    recommendation.nodesToRemove && (
                      <div className="space-y-2">
                        <h4 className="font-medium text-sm">
                          Nodes Recommended for Removal:
                        </h4>
                        <div className="flex flex-wrap gap-2">
                          {recommendation.nodesToRemove.map((nodeName) => (
                            <Badge key={nodeName} variant="outline">
                              {nodeName}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                </CardContent>
              </Card>
            )}

          {/* Nodes Table */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                Cluster Nodes
              </CardTitle>
              <CardDescription>
                {nodes.length} node{nodes.length !== 1 ? "s" : ""} in cluster
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Instance Type</TableHead>
                    <TableHead>Zone</TableHead>
                    <TableHead>CPU Usage</TableHead>
                    <TableHead>Memory Usage</TableHead>
                    <TableHead>Pods</TableHead>
                    <TableHead>Cost/Hour</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {nodes.map((node) => (
                    <TableRow key={node.name}>
                      <TableCell className="font-medium">{node.name}</TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            node.status === "Ready" ? "default" : "destructive"
                          }
                          className="gap-1"
                        >
                          {node.status === "Ready" ? (
                            <CheckCircle2 className="h-3 w-3" />
                          ) : (
                            <AlertTriangle className="h-3 w-3" />
                          )}
                          {node.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div>
                          <div className="font-medium">{node.instanceType}</div>
                          <Badge
                            variant={
                              node.capacityType === "spot"
                                ? "secondary"
                                : "outline"
                            }
                            className="text-xs"
                          >
                            {node.capacityType}
                          </Badge>
                        </div>
                      </TableCell>
                      <TableCell>{node.zone}</TableCell>
                      <TableCell>
                        <div className="space-y-1">
                          <div className="flex items-center justify-between text-sm">
                            <span
                              className={getUtilizationColor(
                                node.cpu.usagePercent
                              )}
                            >
                              {node.cpu.usagePercent}%
                            </span>
                            <span className="text-muted-foreground text-xs">
                              {node.cpu.usage}/{node.cpu.capacity}
                            </span>
                          </div>
                          <Progress value={node.cpu.usagePercent} />
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="space-y-1">
                          <div className="flex items-center justify-between text-sm">
                            <span
                              className={getUtilizationColor(
                                node.memory.usagePercent
                              )}
                            >
                              {node.memory.usagePercent}%
                            </span>
                            <span className="text-muted-foreground text-xs">
                              {node.memory.usage}/{node.memory.capacity}
                            </span>
                          </div>
                          <Progress value={node.memory.usagePercent} />
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">
                          {node.pods.current}/{node.pods.capacity}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="font-medium">
                          ${node.costPerHour.toFixed(3)}
                        </span>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {/* Summary Stats */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">
                  Total Nodes
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{nodes.length}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  {nodes.filter((n) => n.status === "Ready").length} ready
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">
                  Avg CPU Usage
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(
                    nodes.reduce((acc, n) => acc + n.cpu.usagePercent, 0) /
                    nodes.length
                  ).toFixed(0)}
                  %
                </div>
                <Progress
                  value={
                    nodes.reduce((acc, n) => acc + n.cpu.usagePercent, 0) /
                    nodes.length
                  }
                  className="mt-2"
                />
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">
                  Avg Memory Usage
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(
                    nodes.reduce((acc, n) => acc + n.memory.usagePercent, 0) /
                    nodes.length
                  ).toFixed(0)}
                  %
                </div>
                <Progress
                  value={
                    nodes.reduce((acc, n) => acc + n.memory.usagePercent, 0) /
                    nodes.length
                  }
                  className="mt-2"
                />
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">
                  Est. Monthly Cost
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  $
                  {(
                    nodes.reduce((acc, n) => acc + n.costPerHour, 0) *
                    24 *
                    30
                  ).toFixed(2)}
                </div>
                <p className="text-xs text-muted-foreground mt-1">
                  Based on current nodes
                </p>
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  );
};

export default ClusterNodes;
