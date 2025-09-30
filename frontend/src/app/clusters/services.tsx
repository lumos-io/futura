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
import { Input } from "@/components/ui/input";
import { Cluster } from "@/models/kubernetes";
import { CloudProviderConnection } from "@/models/cloud-provider";
import {
  AlertCircle,
  CheckCircle2,
  Database,
  Globe,
  HardDrive,
  Network,
  Search,
  Zap,
} from "lucide-react";

interface ServicesProps {
  title: string;
}

type SLOStatus = "healthy" | "degraded" | "violated";

type ServiceSLO = {
  serviceName: string;
  namespace: string;
  podName: string;
  containerName: string;

  // SLO Targets
  targetP95Latency: number; // ms
  targetErrorRate: number; // percentage
  targetThroughput: number; // rps

  // Current Metrics (eBPF)
  http: {
    requestCount: number;
    errorCount: number;
    errorRate: number;
    latencyP50: number;
    latencyP95: number;
    latencyP99: number;
    throughput: number;
    activeConnections: number;
  };

  memory: {
    allocCount: number;
    allocBytes: number;
    freeCount: number;
    leakSuspected: boolean;
    heapSize: number;
    gcCount: number;
  };

  cpu: {
    usagePercent: number;
    contextSwitches: number;
    runqueueLatency: number;
  };

  network: {
    bytesReceived: number;
    bytesSent: number;
    connectionsActive: number;
    connectionsTotal: number;
  };

  filesystem: {
    readsCount: number;
    writesCount: number;
    readBytes: number;
    writeBytes: number;
    readLatencyAvg: number;
    writeLatencyAvg: number;
  };

  // SLO Health
  sloStatus: SLOStatus;
  sloScore: number; // 0-100
  violations: string[];

  lastUpdated: number; // unix timestamp
};

const ClusterServices: React.FC<ServicesProps> = ({ title }) => {
  const { user } = useAuth();

  const [providersConnection, setProvidersConnection] = useState<
    CloudProviderConnection[]
  >([]);
  const [selectedProvider, setSelectedProvider] =
    useState<CloudProviderConnection>();
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [selectedCluster, setSelectedCluster] = useState<Cluster>();
  const [services, setServices] = useState<ServiceSLO[]>([]);
  const [filteredServices, setFilteredServices] = useState<ServiceSLO[]>([]);
  const [selectedService, setSelectedService] = useState<ServiceSLO | null>(
    null
  );
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const orgId = user?.organizationId;

  // Fetch providers
  useEffect(() => {
    const fetchProviders = async () => {
      if (!orgId) return;

      const res = await fetch(`/api/organizations/${orgId}/connects`);
      if (!res.ok) return;

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

  // Fetch clusters
  useEffect(() => {
    if (!selectedProvider || !orgId) return;

    const fetchClusters = async () => {
      setLoading(true);

      try {
        const res = await fetch(
          `/api/organizations/${orgId}/connects/${selectedProvider.id}/clusters`
        );
        if (!res.ok) throw new Error("Failed to fetch clusters");

        const data = await res.json();
        const fetchedClusters: Cluster[] = data.data ?? [];
        setClusters(fetchedClusters);
        setSelectedCluster(fetchedClusters[0]);
      } catch (err) {
        console.error(err);
        setClusters([]);
      } finally {
        setLoading(false);
      }
    };

    fetchClusters();
  }, [selectedProvider, orgId]);

  // Fetch service SLOs and eBPF metrics
  useEffect(() => {
    if (!selectedCluster || !orgId) return;

    const fetchServices = async () => {
      setLoading(true);

      try {
        const res = await fetch(
          `/api/organizations/${orgId}/clusters/${selectedCluster.id}/slo-metrics`
        );
        if (!res.ok) throw new Error("Failed to fetch SLO metrics");

        const data = await res.json();
        setServices(data.data);
        setFilteredServices(data.data);

        if (data.data.length > 0) {
          setSelectedService(data.data[0]);
        }
      } catch (err) {
        console.error("Error fetching SLO metrics:", err);
        // Mock data for development
        setServices(mockServices);
        setFilteredServices(mockServices);
        setSelectedService(mockServices[0]);
      } finally {
        setLoading(false);
      }
    };

    fetchServices();

    // Poll every 5 seconds for real-time updates
    const interval = setInterval(fetchServices, 5000);
    return () => clearInterval(interval);
  }, [selectedCluster, orgId]);

  // Filter services
  useEffect(() => {
    let filtered = services;

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (s) =>
          s.serviceName.toLowerCase().includes(query) ||
          s.namespace.toLowerCase().includes(query) ||
          s.podName.toLowerCase().includes(query)
      );
    }

    if (statusFilter !== "all") {
      filtered = filtered.filter((s) => s.sloStatus === statusFilter);
    }

    setFilteredServices(filtered);
  }, [searchQuery, statusFilter, services]);

  const getSLOStatusBadge = (status: SLOStatus) => {
    switch (status) {
      case "healthy":
        return (
          <Badge className="bg-green-500 text-white">
            <CheckCircle2 className="h-3 w-3 mr-1" />
            Healthy
          </Badge>
        );
      case "degraded":
        return (
          <Badge className="bg-yellow-500 text-white">
            <AlertCircle className="h-3 w-3 mr-1" />
            Degraded
          </Badge>
        );
      case "violated":
        return (
          <Badge variant="destructive">
            <AlertCircle className="h-3 w-3 mr-1" />
            Violated
          </Badge>
        );
    }
  };

  const getSLOScoreColor = (score: number) => {
    if (score >= 95) return "text-green-500";
    if (score >= 85) return "text-yellow-500";
    return "text-red-500";
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const formatNumber = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(2)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(2)}K`;
    return num.toString();
  };

  const getServiceStats = () => {
    return {
      total: filteredServices.length,
      healthy: filteredServices.filter((s) => s.sloStatus === "healthy").length,
      degraded: filteredServices.filter((s) => s.sloStatus === "degraded")
        .length,
      violated: filteredServices.filter((s) => s.sloStatus === "violated")
        .length,
      avgScore:
        filteredServices.reduce((acc, s) => acc + s.sloScore, 0) /
        (filteredServices.length || 1),
    };
  };

  const stats = getServiceStats();

  return (
    <div className="p-10 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-semibold">{title}</h1>
          <p className="text-muted-foreground mt-1">
            Monitor Service Level Objectives and eBPF application metrics in
            real-time
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col md:flex-row gap-4">
        <div className="w-full md:w-60">
          <Select
            value={selectedProvider?.id.toString() ?? ""}
            onValueChange={(value) => {
              const found = providersConnection.find(
                (p) => p.id.toString() === value
              );
              if (found) setSelectedProvider(found);
            }}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select Provider" />
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
              if (found) setSelectedCluster(found);
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

        <div className="flex-1">
          <div className="relative">
            <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search services..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>

        <div className="w-full md:w-40">
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Status</SelectItem>
              <SelectItem value="healthy">Healthy</SelectItem>
              <SelectItem value="degraded">Degraded</SelectItem>
              <SelectItem value="violated">Violated</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Stats Overview */}
      {!loading && selectedCluster && (
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">
                Total Services
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.total}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4 text-green-500" />
                Healthy
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-500">
                {stats.healthy}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <AlertCircle className="h-4 w-4 text-yellow-500" />
                Degraded
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-yellow-500">
                {stats.degraded}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <AlertCircle className="h-4 w-4 text-red-500" />
                Violated
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-red-500">
                {stats.violated}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Avg SLO Score</CardTitle>
            </CardHeader>
            <CardContent>
              <div
                className={`text-2xl font-bold ${getSLOScoreColor(stats.avgScore)}`}
              >
                {stats.avgScore.toFixed(1)}%
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Loading State */}
      {loading && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          {[...Array(6)].map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-4 w-32" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-20 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Services Grid */}
      {!loading && selectedCluster && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          {/* Service List */}
          <div className="lg:col-span-1 space-y-3">
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Services</CardTitle>
                <CardDescription>
                  {filteredServices.length} service(s) found
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-2 max-h-[600px] overflow-y-auto">
                {filteredServices.map((service) => (
                  <Card
                    key={`${service.namespace}-${service.serviceName}`}
                    className={`cursor-pointer transition-all hover:shadow-md ${
                      selectedService?.serviceName === service.serviceName &&
                      selectedService?.namespace === service.namespace
                        ? "ring-2 ring-primary"
                        : ""
                    }`}
                    onClick={() => setSelectedService(service)}
                  >
                    <CardHeader className="pb-3">
                      <div className="flex items-center justify-between">
                        <div className="flex-1 min-w-0">
                          <CardTitle className="text-sm truncate">
                            {service.serviceName}
                          </CardTitle>
                          <CardDescription className="text-xs">
                            {service.namespace}
                          </CardDescription>
                        </div>
                        {getSLOStatusBadge(service.sloStatus)}
                      </div>
                    </CardHeader>
                    <CardContent className="pt-0">
                      <div className="space-y-1 text-xs">
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">P95:</span>
                          <span className="font-medium">
                            {service.http.latencyP95.toFixed(1)}ms
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Errors:</span>
                          <span className="font-medium">
                            {service.http.errorRate.toFixed(2)}%
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">RPS:</span>
                          <span className="font-medium">
                            {formatNumber(service.http.throughput)}
                          </span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}

                {filteredServices.length === 0 && (
                  <div className="text-center py-8 text-muted-foreground">
                    No services found
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Service Details */}
          {selectedService && (
            <div className="lg:col-span-2 space-y-4">
              {/* Service Header */}
              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-xl">
                        {selectedService.serviceName}
                      </CardTitle>
                      <CardDescription>
                        {selectedService.namespace} • {selectedService.podName}
                      </CardDescription>
                    </div>
                    <div className="flex items-center gap-3">
                      {getSLOStatusBadge(selectedService.sloStatus)}
                      <div className="text-right">
                        <div
                          className={`text-2xl font-bold ${getSLOScoreColor(selectedService.sloScore)}`}
                        >
                          {selectedService.sloScore}%
                        </div>
                        <div className="text-xs text-muted-foreground">
                          SLO Score
                        </div>
                      </div>
                    </div>
                  </div>
                </CardHeader>
                {selectedService.violations.length > 0 && (
                  <CardContent className="pt-0">
                    <div className="bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-lg p-3">
                      <div className="flex items-start gap-2">
                        <AlertCircle className="h-4 w-4 text-red-500 mt-0.5" />
                        <div>
                          <div className="font-medium text-sm text-red-900 dark:text-red-100">
                            SLO Violations
                          </div>
                          <ul className="text-xs text-red-700 dark:text-red-300 mt-1 space-y-1">
                            {selectedService.violations.map((v, i) => (
                              <li key={i}>• {v}</li>
                            ))}
                          </ul>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                )}
              </Card>

              {/* HTTP Metrics */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Globe className="h-5 w-5" />
                    HTTP Performance
                  </CardTitle>
                </CardHeader>
                <CardContent className="grid grid-cols-2 gap-4">
                  <div>
                    <div className="text-sm text-muted-foreground">
                      Request Count
                    </div>
                    <div className="text-2xl font-bold">
                      {formatNumber(selectedService.http.requestCount)}
                    </div>
                  </div>
                  <div>
                    <div className="text-sm text-muted-foreground">
                      Error Rate
                    </div>
                    <div className="text-2xl font-bold">
                      {selectedService.http.errorRate.toFixed(2)}%
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Target: ≤ {selectedService.targetErrorRate}%
                    </div>
                  </div>
                  <div>
                    <div className="text-sm text-muted-foreground">P50 Latency</div>
                    <div className="text-xl font-bold">
                      {selectedService.http.latencyP50.toFixed(1)}ms
                    </div>
                  </div>
                  <div>
                    <div className="text-sm text-muted-foreground">P95 Latency</div>
                    <div className="text-xl font-bold">
                      {selectedService.http.latencyP95.toFixed(1)}ms
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Target: ≤ {selectedService.targetP95Latency}ms
                    </div>
                  </div>
                  <div>
                    <div className="text-sm text-muted-foreground">P99 Latency</div>
                    <div className="text-xl font-bold">
                      {selectedService.http.latencyP99.toFixed(1)}ms
                    </div>
                  </div>
                  <div>
                    <div className="text-sm text-muted-foreground">Throughput</div>
                    <div className="text-xl font-bold">
                      {formatNumber(selectedService.http.throughput)} RPS
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Target: ≥ {formatNumber(selectedService.targetThroughput)}{" "}
                      RPS
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Resource Metrics */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Memory */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-base">
                      <Database className="h-4 w-4" />
                      Memory
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Heap Size:</span>
                      <span className="font-medium">
                        {formatBytes(selectedService.memory.heapSize)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Allocations:</span>
                      <span className="font-medium">
                        {formatNumber(selectedService.memory.allocCount)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">GC Count:</span>
                      <span className="font-medium">
                        {selectedService.memory.gcCount}
                      </span>
                    </div>
                    {selectedService.memory.leakSuspected && (
                      <Badge variant="destructive" className="w-full justify-center">
                        Memory Leak Suspected
                      </Badge>
                    )}
                  </CardContent>
                </Card>

                {/* CPU */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-base">
                      <Zap className="h-4 w-4" />
                      CPU
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Usage:</span>
                      <span className="font-medium">
                        {selectedService.cpu.usagePercent.toFixed(1)}%
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">
                        Context Switches:
                      </span>
                      <span className="font-medium">
                        {formatNumber(selectedService.cpu.contextSwitches)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">
                        Runqueue Latency:
                      </span>
                      <span className="font-medium">
                        {selectedService.cpu.runqueueLatency.toFixed(2)}ms
                      </span>
                    </div>
                  </CardContent>
                </Card>

                {/* Network */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-base">
                      <Network className="h-4 w-4" />
                      Network
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Received:</span>
                      <span className="font-medium">
                        {formatBytes(selectedService.network.bytesReceived)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Sent:</span>
                      <span className="font-medium">
                        {formatBytes(selectedService.network.bytesSent)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Connections:</span>
                      <span className="font-medium">
                        {selectedService.network.connectionsActive} active
                      </span>
                    </div>
                  </CardContent>
                </Card>

                {/* Filesystem */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-base">
                      <HardDrive className="h-4 w-4" />
                      Filesystem
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Reads:</span>
                      <span className="font-medium">
                        {formatNumber(selectedService.filesystem.readsCount)} (
                        {formatBytes(selectedService.filesystem.readBytes)})
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Writes:</span>
                      <span className="font-medium">
                        {formatNumber(selectedService.filesystem.writesCount)} (
                        {formatBytes(selectedService.filesystem.writeBytes)})
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Avg Latency:</span>
                      <span className="font-medium">
                        R: {selectedService.filesystem.readLatencyAvg.toFixed(2)}ms
                        / W: {selectedService.filesystem.writeLatencyAvg.toFixed(2)}
                        ms
                      </span>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

// Mock data for development
const mockServices: ServiceSLO[] = [
  {
    serviceName: "api-gateway",
    namespace: "production",
    podName: "api-gateway-7d8f9c-xyz",
    containerName: "gateway",
    targetP95Latency: 250,
    targetErrorRate: 1,
    targetThroughput: 1000,
    http: {
      requestCount: 45230,
      errorCount: 127,
      errorRate: 0.28,
      latencyP50: 52.3,
      latencyP95: 187.5,
      latencyP99: 342.1,
      throughput: 1247,
      activeConnections: 234,
    },
    memory: {
      allocCount: 1523400,
      allocBytes: 5242880000,
      freeCount: 1521200,
      leakSuspected: false,
      heapSize: 2147483648,
      gcCount: 142,
    },
    cpu: {
      usagePercent: 34.7,
      contextSwitches: 45230,
      runqueueLatency: 1.23,
    },
    network: {
      bytesReceived: 1073741824,
      bytesSent: 2147483648,
      connectionsActive: 234,
      connectionsTotal: 12450,
    },
    filesystem: {
      readsCount: 8420,
      writesCount: 3210,
      readBytes: 524288000,
      writeBytes: 209715200,
      readLatencyAvg: 2.34,
      writeLatencyAvg: 5.67,
    },
    sloStatus: "healthy",
    sloScore: 98.5,
    violations: [],
    lastUpdated: Math.floor(Date.now() / 1000),
  },
  {
    serviceName: "user-service",
    namespace: "production",
    podName: "user-service-6c7d8e-abc",
    containerName: "service",
    targetP95Latency: 150,
    targetErrorRate: 0.5,
    targetThroughput: 500,
    http: {
      requestCount: 28450,
      errorCount: 450,
      errorRate: 1.58,
      latencyP50: 95.2,
      latencyP95: 234.8,
      latencyP99: 512.3,
      throughput: 623,
      activeConnections: 145,
    },
    memory: {
      allocCount: 2340500,
      allocBytes: 8589934592,
      freeCount: 2335100,
      leakSuspected: true,
      heapSize: 4294967296,
      gcCount: 287,
    },
    cpu: {
      usagePercent: 67.3,
      contextSwitches: 89340,
      runqueueLatency: 4.56,
    },
    network: {
      bytesReceived: 536870912,
      bytesSent: 1073741824,
      connectionsActive: 145,
      connectionsTotal: 8920,
    },
    filesystem: {
      readsCount: 12340,
      writesCount: 5670,
      readBytes: 1048576000,
      writeBytes: 524288000,
      readLatencyAvg: 3.45,
      writeLatencyAvg: 8.92,
    },
    sloStatus: "degraded",
    sloScore: 87.2,
    violations: ["P95 latency exceeds target", "Error rate above threshold"],
    lastUpdated: Math.floor(Date.now() / 1000),
  },
  {
    serviceName: "payment-processor",
    namespace: "production",
    podName: "payment-processor-5b6c7d-def",
    containerName: "processor",
    targetP95Latency: 500,
    targetErrorRate: 0.1,
    targetThroughput: 200,
    http: {
      requestCount: 15670,
      errorCount: 523,
      errorRate: 3.34,
      latencyP50: 234.5,
      latencyP95: 789.3,
      latencyP99: 1245.7,
      throughput: 342,
      activeConnections: 89,
    },
    memory: {
      allocCount: 1890400,
      allocBytes: 6442450944,
      freeCount: 1887200,
      leakSuspected: false,
      heapSize: 3221225472,
      gcCount: 198,
    },
    cpu: {
      usagePercent: 52.8,
      contextSwitches: 67890,
      runqueueLatency: 2.89,
    },
    network: {
      bytesReceived: 268435456,
      bytesSent: 536870912,
      connectionsActive: 89,
      connectionsTotal: 5430,
    },
    filesystem: {
      readsCount: 23450,
      writesCount: 18920,
      readBytes: 2097152000,
      writeBytes: 1572864000,
      readLatencyAvg: 5.67,
      writeLatencyAvg: 12.34,
    },
    sloStatus: "violated",
    sloScore: 72.8,
    violations: [
      "P95 latency significantly exceeds target",
      "Error rate 33x above threshold",
      "Throughput below target",
    ],
    lastUpdated: Math.floor(Date.now() / 1000),
  },
];

export default ClusterServices;
