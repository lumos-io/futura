import React, { useEffect, useState, useRef } from "react";
import { useAuth } from "@/hooks/auth-provider";
import { useSSE } from "@/hooks/sse-handler";
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
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
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
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Area,
  AreaChart,
} from "recharts";

interface ServicesProps {
  title: string;
}

type SLOStatus = "healthy" | "degraded" | "violated";

type MetricHistoryPoint = {
  time: string;
  value: number;
};

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
    history: MetricHistoryPoint[]; // MB over time
  };

  cpu: {
    usagePercent: number;
    contextSwitches: number;
    runqueueLatency: number;
    history: MetricHistoryPoint[]; // percentage over time
  };

  network: {
    bytesReceived: number;
    bytesSent: number;
    connectionsActive: number;
    connectionsTotal: number;
    history: { time: string; received: number; sent: number }[]; // MB/s over time
  };

  filesystem: {
    readsCount: number;
    writesCount: number;
    readBytes: number;
    writeBytes: number;
    readLatencyAvg: number;
    writeLatencyAvg: number;
    history: { time: string; reads: number; writes: number }[]; // MB/s over time
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
  const [sheetOpen, setSheetOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage] = useState(10);
  const selectedServiceIdRef = useRef<string | null>(null);

  const orgId = user?.organizationId;

  // SSE URL for real-time SLO metrics
  const sseUrl = selectedCluster && orgId
    ? `/api/organizations/${orgId}/clusters/${selectedCluster.id}/slo-metrics`
    : "";

  // Use SSE hook for real-time updates
  const { latest: sseData } = useSSE<{ data: ServiceSLO[] }>(sseUrl, {
    event: "slo-metrics",
  });

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

  // Handle SSE data updates
  useEffect(() => {
    if (!selectedCluster) return;

    // Use mock data for now since backend returns empty array
    // TODO: Remove mock data when backend implements real SLO metrics
    setLoading(true);
    setServices(mockServices);
    setFilteredServices(mockServices);

    // Set initial selected service if none selected and sheet is not open
    if (!selectedServiceIdRef.current && mockServices.length > 0) {
      setSelectedService(mockServices[0]);
      selectedServiceIdRef.current = `${mockServices[0].namespace}-${mockServices[0].serviceName}`;
    }

    setLoading(false);
  }, [selectedCluster]);

  // Update services when SSE data arrives (keeping selected service)
  useEffect(() => {
    if (!sseData || !sseData.data) return;

    // If backend sends real data, use it
    if (sseData.data.length > 0) {
      setServices(sseData.data);

      // Preserve selected service across updates
      if (selectedServiceIdRef.current) {
        const [namespace, serviceName] = selectedServiceIdRef.current.split("-");
        const updatedSelectedService = sseData.data.find(
          (s) => s.namespace === namespace && s.serviceName === serviceName
        );
        if (updatedSelectedService) {
          setSelectedService(updatedSelectedService);
        }
      }
    }
  }, [sseData]);

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
    setCurrentPage(1); // Reset to first page when filters change
  }, [searchQuery, statusFilter, services]);

  // Pagination
  const totalPages = Math.ceil(filteredServices.length / itemsPerPage);
  const startIndex = (currentPage - 1) * itemsPerPage;
  const endIndex = startIndex + itemsPerPage;
  const paginatedServices = filteredServices.slice(startIndex, endIndex);

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
              <CardTitle className="text-sm font-medium">
                Avg SLO Score
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div
                className={`text-2xl font-bold ${getSLOScoreColor(
                  stats.avgScore
                )}`}
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

      {/* Services Table */}
      {!loading && selectedCluster && (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-lg">Services</CardTitle>
                  <CardDescription>
                    Showing {startIndex + 1}-
                    {Math.min(endIndex, filteredServices.length)} of{" "}
                    {filteredServices.length} service(s)
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {/* Table */}
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left py-3 px-4 font-medium text-sm">
                        Service
                      </th>
                      <th className="text-left py-3 px-4 font-medium text-sm">
                        Namespace
                      </th>
                      <th className="text-left py-3 px-4 font-medium text-sm">
                        Status
                      </th>
                      <th className="text-right py-3 px-4 font-medium text-sm">
                        SLO Score
                      </th>
                      <th className="text-right py-3 px-4 font-medium text-sm">
                        P95 Latency
                      </th>
                      <th className="text-right py-3 px-4 font-medium text-sm">
                        Error Rate
                      </th>
                      <th className="text-right py-3 px-4 font-medium text-sm">
                        Throughput
                      </th>
                      <th className="text-left py-3 px-4 font-medium text-sm">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {paginatedServices.map((service) => (
                      <tr
                        key={`${service.namespace}-${service.serviceName}`}
                        className={`border-b hover:bg-muted/50 transition-colors ${
                          selectedService?.serviceName ===
                            service.serviceName &&
                          selectedService?.namespace === service.namespace
                            ? "bg-muted"
                            : ""
                        }`}
                      >
                        <td className="py-3 px-4">
                          <div className="font-medium">
                            {service.serviceName}
                          </div>
                          <div className="text-xs text-muted-foreground truncate max-w-[200px]">
                            {service.podName}
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <Badge variant="outline">{service.namespace}</Badge>
                        </td>
                        <td className="py-3 px-4">
                          {getSLOStatusBadge(service.sloStatus)}
                        </td>
                        <td className="py-3 px-4 text-right">
                          <span
                            className={`font-bold ${getSLOScoreColor(
                              service.sloScore
                            )}`}
                          >
                            {service.sloScore.toFixed(1)}%
                          </span>
                        </td>
                        <td className="py-3 px-4 text-right">
                          <div className="font-medium">
                            {service.http.latencyP95.toFixed(1)}ms
                          </div>
                          <div className="text-xs text-muted-foreground">
                            Target: ≤{service.targetP95Latency}ms
                          </div>
                        </td>
                        <td className="py-3 px-4 text-right">
                          <div className="font-medium">
                            {service.http.errorRate.toFixed(2)}%
                          </div>
                          <div className="text-xs text-muted-foreground">
                            Target: ≤{service.targetErrorRate}%
                          </div>
                        </td>
                        <td className="py-3 px-4 text-right">
                          <div className="font-medium">
                            {formatNumber(service.http.throughput)} RPS
                          </div>
                          <div className="text-xs text-muted-foreground">
                            Target: ≥{formatNumber(service.targetThroughput)}
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <button
                            onClick={() => {
                              setSelectedService(service);
                              selectedServiceIdRef.current = `${service.namespace}-${service.serviceName}`;
                              setSheetOpen(true);
                            }}
                            className="text-sm text-primary hover:underline"
                          >
                            View Details
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>

                {filteredServices.length === 0 && (
                  <div className="text-center py-12 text-muted-foreground">
                    No services found matching your filters
                  </div>
                )}
              </div>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4 pt-4 border-t">
                  <div className="text-sm text-muted-foreground">
                    Page {currentPage} of {totalPages}
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() =>
                        setCurrentPage((prev) => Math.max(1, prev - 1))
                      }
                      disabled={currentPage === 1}
                      className="px-3 py-1 text-sm border rounded-md hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Previous
                    </button>

                    {/* Page numbers */}
                    <div className="flex gap-1">
                      {Array.from(
                        { length: Math.min(5, totalPages) },
                        (_, i) => {
                          let pageNum;
                          if (totalPages <= 5) {
                            pageNum = i + 1;
                          } else if (currentPage <= 3) {
                            pageNum = i + 1;
                          } else if (currentPage >= totalPages - 2) {
                            pageNum = totalPages - 4 + i;
                          } else {
                            pageNum = currentPage - 2 + i;
                          }

                          return (
                            <button
                              key={pageNum}
                              onClick={() => setCurrentPage(pageNum)}
                              className={`px-3 py-1 text-sm border rounded-md hover:bg-muted ${
                                currentPage === pageNum
                                  ? "bg-primary text-primary-foreground"
                                  : ""
                              }`}
                            >
                              {pageNum}
                            </button>
                          );
                        }
                      )}
                    </div>

                    <button
                      onClick={() =>
                        setCurrentPage((prev) => Math.min(totalPages, prev + 1))
                      }
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 text-sm border rounded-md hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Next
                    </button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {/* Service Details Sheet */}
      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <SheetContent className="w-full sm:max-w-3xl overflow-y-auto p-6">
          {selectedService && (
            <div className="space-y-6">
              {/* Service Header */}
              <div className="space-y-2">
                <SheetHeader>
                  <SheetTitle className="text-2xl font-bold">
                    {selectedService.serviceName}
                  </SheetTitle>
                  <SheetDescription className="text-base">
                    {selectedService.namespace} • {selectedService.podName}
                  </SheetDescription>
                </SheetHeader>
                <div className="flex items-center gap-4 pt-2">
                  {getSLOStatusBadge(selectedService.sloStatus)}
                  <div className="flex items-baseline gap-2">
                    <div
                      className={`text-3xl font-bold ${getSLOScoreColor(
                        selectedService.sloScore
                      )}`}
                    >
                      {selectedService.sloScore.toFixed(1)}%
                    </div>
                    <div className="text-sm text-muted-foreground">
                      SLO Score
                    </div>
                  </div>
                </div>
              </div>

              {selectedService.violations.length > 0 && (
                <div className="bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <AlertCircle className="h-5 w-5 text-red-500 mt-0.5" />
                    <div>
                      <div className="font-semibold text-red-900 dark:text-red-100">
                        SLO Violations
                      </div>
                      <ul className="text-sm text-red-700 dark:text-red-300 mt-2 space-y-1">
                        {selectedService.violations.map((v, i) => (
                          <li key={i}>• {v}</li>
                        ))}
                      </ul>
                    </div>
                  </div>
                </div>
              )}

              {/* HTTP Performance Summary */}
              <div className="border rounded-lg p-4 space-y-3">
                <div className="flex items-center gap-2 pb-2 border-b">
                  <Globe className="h-5 w-5" />
                  <h3 className="font-semibold text-lg">HTTP Performance</h3>
                </div>
                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <div className="text-xs text-muted-foreground mb-1">
                      P95 Latency
                    </div>
                    <div className="text-xl font-bold">
                      {selectedService.http.latencyP95.toFixed(1)}ms
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Target: ≤{selectedService.targetP95Latency}ms
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground mb-1">
                      Error Rate
                    </div>
                    <div className="text-xl font-bold">
                      {selectedService.http.errorRate.toFixed(2)}%
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Target: ≤{selectedService.targetErrorRate}%
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground mb-1">
                      Throughput
                    </div>
                    <div className="text-xl font-bold">
                      {formatNumber(selectedService.http.throughput)} RPS
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Target: ≥{formatNumber(selectedService.targetThroughput)}
                    </div>
                  </div>
                </div>
              </div>

              {/* Memory Usage Chart */}
              <div className="border rounded-lg p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Database className="h-5 w-5" />
                    <h3 className="font-semibold">Memory Usage</h3>
                  </div>
                  <div className="text-right">
                    <div className="text-lg font-bold">
                      {formatBytes(selectedService.memory.heapSize)}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Heap Size
                    </div>
                  </div>
                </div>
                <ResponsiveContainer width="100%" height={180}>
                  <AreaChart data={selectedService.memory.history}>
                    <defs>
                      <linearGradient id="memoryGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                    <XAxis
                      dataKey="time"
                      stroke="#9ca3af"
                      style={{ fontSize: '12px' }}
                    />
                    <YAxis
                      stroke="#9ca3af"
                      style={{ fontSize: '12px' }}
                      tickFormatter={(value) => `${value}MB`}
                    />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#1f2937',
                        border: '1px solid #374151',
                        borderRadius: '6px'
                      }}
                      labelStyle={{ color: '#f3f4f6' }}
                      formatter={(value: number) => [`${value.toFixed(1)}MB`, 'Memory']}
                    />
                    <Area
                      type="monotone"
                      dataKey="value"
                      stroke="#3b82f6"
                      strokeWidth={2}
                      fill="url(#memoryGradient)"
                    />
                  </AreaChart>
                </ResponsiveContainer>
                <div className="grid grid-cols-3 gap-4 text-sm pt-2 border-t">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Allocs:</span>
                    <span className="font-medium">
                      {formatNumber(selectedService.memory.allocCount)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">GC:</span>
                    <span className="font-medium">
                      {selectedService.memory.gcCount}
                    </span>
                  </div>
                  {selectedService.memory.leakSuspected && (
                    <Badge variant="destructive" className="text-xs">
                      Leak Suspected
                    </Badge>
                  )}
                </div>
              </div>

              {/* CPU Usage Chart */}
              <div className="border rounded-lg p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Zap className="h-5 w-5" />
                    <h3 className="font-semibold">CPU Usage</h3>
                  </div>
                  <div className="text-right">
                    <div className="text-lg font-bold">
                      {selectedService.cpu.usagePercent.toFixed(1)}%
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Current
                    </div>
                  </div>
                </div>
                <ResponsiveContainer width="100%" height={180}>
                  <AreaChart data={selectedService.cpu.history}>
                    <defs>
                      <linearGradient id="cpuGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.3}/>
                        <stop offset="95%" stopColor="#f59e0b" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                    <XAxis
                      dataKey="time"
                      stroke="#9ca3af"
                      style={{ fontSize: '12px' }}
                    />
                    <YAxis
                      stroke="#9ca3af"
                      style={{ fontSize: '12px' }}
                      tickFormatter={(value) => `${value}%`}
                    />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#1f2937',
                        border: '1px solid #374151',
                        borderRadius: '6px'
                      }}
                      labelStyle={{ color: '#f3f4f6' }}
                      formatter={(value: number) => [`${value.toFixed(1)}%`, 'CPU']}
                    />
                    <Area
                      type="monotone"
                      dataKey="value"
                      stroke="#f59e0b"
                      strokeWidth={2}
                      fill="url(#cpuGradient)"
                    />
                  </AreaChart>
                </ResponsiveContainer>
                <div className="grid grid-cols-2 gap-4 text-sm pt-2 border-t">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Context Switches:</span>
                    <span className="font-medium">
                      {formatNumber(selectedService.cpu.contextSwitches)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Runqueue:</span>
                    <span className="font-medium">
                      {selectedService.cpu.runqueueLatency.toFixed(2)}ms
                    </span>
                  </div>
                </div>
              </div>

              {/* Network Traffic Chart */}
              <div className="border rounded-lg p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Network className="h-5 w-5" />
                    <h3 className="font-semibold">Network Traffic</h3>
                  </div>
                  <div className="text-right">
                    <div className="text-sm font-medium">
                      {selectedService.network.connectionsActive} active
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Connections
                    </div>
                  </div>
                </div>
                <ResponsiveContainer width="100%" height={180}>
                  <LineChart data={selectedService.network.history}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                    <XAxis
                      dataKey="time"
                      stroke="#9ca3af"
                      style={{ fontSize: '12px' }}
                    />
                    <YAxis
                      stroke="#9ca3af"
                      style={{ fontSize: '12px' }}
                      tickFormatter={(value) => `${value}MB/s`}
                    />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#1f2937',
                        border: '1px solid #374151',
                        borderRadius: '6px'
                      }}
                      labelStyle={{ color: '#f3f4f6' }}
                      formatter={(value: number, name: string) => [
                        `${value.toFixed(2)}MB/s`,
                        name === 'received' ? 'Received' : 'Sent'
                      ]}
                    />
                    <Line
                      type="monotone"
                      dataKey="received"
                      stroke="#10b981"
                      strokeWidth={2}
                      dot={false}
                    />
                    <Line
                      type="monotone"
                      dataKey="sent"
                      stroke="#3b82f6"
                      strokeWidth={2}
                      dot={false}
                    />
                  </LineChart>
                </ResponsiveContainer>
                <div className="grid grid-cols-2 gap-4 text-sm pt-2 border-t">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Total Received:</span>
                    <span className="font-medium">
                      {formatBytes(selectedService.network.bytesReceived)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Total Sent:</span>
                    <span className="font-medium">
                      {formatBytes(selectedService.network.bytesSent)}
                    </span>
                  </div>
                </div>
              </div>

              {/* Filesystem I/O Chart */}
              <div className="border rounded-lg p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <HardDrive className="h-5 w-5" />
                    <h3 className="font-semibold">Filesystem I/O</h3>
                  </div>
                  <div className="text-right">
                    <div className="text-sm font-medium">
                      R: {selectedService.filesystem.readLatencyAvg.toFixed(1)}ms /
                      W: {selectedService.filesystem.writeLatencyAvg.toFixed(1)}ms
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Avg Latency
                    </div>
                  </div>
                </div>
                <ResponsiveContainer width="100%" height={180}>
                  <LineChart data={selectedService.filesystem.history}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                    <XAxis
                      dataKey="time"
                      stroke="#9ca3af"
                      style={{ fontSize: '12px' }}
                    />
                    <YAxis
                      stroke="#9ca3af"
                      style={{ fontSize: '12px' }}
                      tickFormatter={(value) => `${value}MB/s`}
                    />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#1f2937',
                        border: '1px solid #374151',
                        borderRadius: '6px'
                      }}
                      labelStyle={{ color: '#f3f4f6' }}
                      formatter={(value: number, name: string) => [
                        `${value.toFixed(2)}MB/s`,
                        name === 'reads' ? 'Reads' : 'Writes'
                      ]}
                    />
                    <Line
                      type="monotone"
                      dataKey="reads"
                      stroke="#8b5cf6"
                      strokeWidth={2}
                      dot={false}
                    />
                    <Line
                      type="monotone"
                      dataKey="writes"
                      stroke="#ec4899"
                      strokeWidth={2}
                      dot={false}
                    />
                  </LineChart>
                </ResponsiveContainer>
                <div className="grid grid-cols-2 gap-4 text-sm pt-2 border-t">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Total Read:</span>
                    <span className="font-medium">
                      {formatBytes(selectedService.filesystem.readBytes)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Total Write:</span>
                    <span className="font-medium">
                      {formatBytes(selectedService.filesystem.writeBytes)}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
};

// Helper to generate historical data points
const generateHistoryData = (points: number, baseValue: number, variance: number) => {
  const history: MetricHistoryPoint[] = [];
  const now = Date.now();

  for (let i = points; i >= 0; i--) {
    const time = new Date(now - i * 10000); // 10 second intervals
    const hours = time.getHours();
    const minutes = time.getMinutes();
    const timeStr = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;

    // Add some realistic variation
    const value = baseValue + (Math.random() - 0.5) * variance;
    history.push({
      time: timeStr,
      value: Math.max(0, value),
    });
  }

  return history;
};

// Helper to generate mock services
const generateMockService = (
  index: number,
  namespace: string,
  status: SLOStatus
): ServiceSLO => {
  const serviceNames = [
    "api-gateway",
    "user-service",
    "payment-processor",
    "auth-service",
    "notification-service",
    "email-service",
    "analytics-service",
    "search-service",
    "recommendation-engine",
    "inventory-service",
    "order-service",
    "shipping-service",
    "billing-service",
    "fraud-detection",
    "audit-logger",
    "metrics-collector",
    "cache-service",
    "session-manager",
    "file-storage",
    "image-processor",
    "video-transcoder",
    "webhook-handler",
    "scheduler-service",
    "queue-manager",
    "data-pipeline",
    "ml-inference",
    "feature-flags",
    "config-service",
    "monitoring-agent",
    "logging-service",
    "tracing-collector",
    "secret-manager",
    "backup-service",
    "restore-service",
    "migration-tool",
    "health-checker",
  ];

  const serviceName = `${serviceNames[index % serviceNames.length]}-${
    Math.floor(index / serviceNames.length) || ""
  }`;

  const baseLatency =
    status === "healthy" ? 50 : status === "degraded" ? 150 : 350;
  const baseError =
    status === "healthy" ? 0.1 : status === "degraded" ? 0.8 : 2.5;
  const baseScore = status === "healthy" ? 95 : status === "degraded" ? 85 : 70;

  // Base values for metrics
  const baseCpuUsage = status === "healthy" ? 45 : status === "degraded" ? 65 : 85;
  const baseMemoryMB = Math.floor(Math.random() * 1024) + 512; // 512-1536 MB
  const baseNetworkMB = Math.random() * 10 + 2; // 2-12 MB/s
  const baseFsIO = Math.random() * 5 + 1; // 1-6 MB/s

  // Generate historical data (last 30 data points = ~5 minutes at 10s intervals)
  const memoryHistory = generateHistoryData(30, baseMemoryMB, baseMemoryMB * 0.15);
  const cpuHistory = generateHistoryData(30, baseCpuUsage, 15);

  const networkHistory = memoryHistory.map((point) => ({
    time: point.time,
    received: Math.max(0, baseNetworkMB + (Math.random() - 0.5) * baseNetworkMB * 0.3),
    sent: Math.max(0, baseNetworkMB * 0.8 + (Math.random() - 0.5) * baseNetworkMB * 0.3),
  }));

  const filesystemHistory = memoryHistory.map((point) => ({
    time: point.time,
    reads: Math.max(0, baseFsIO + (Math.random() - 0.5) * baseFsIO * 0.4),
    writes: Math.max(0, baseFsIO * 0.6 + (Math.random() - 0.5) * baseFsIO * 0.4),
  }));

  return {
    serviceName,
    namespace,
    podName: `${serviceName}-${Math.random().toString(36).substr(2, 9)}`,
    containerName: serviceName.split("-")[0],
    targetP95Latency: 250,
    targetErrorRate: 1,
    targetThroughput: 500,
    http: {
      requestCount: Math.floor(Math.random() * 50000) + 10000,
      errorCount: Math.floor(Math.random() * 500) + 10,
      errorRate: baseError + Math.random() * 0.5,
      latencyP50: baseLatency + Math.random() * 50,
      latencyP95: baseLatency * 2 + Math.random() * 100,
      latencyP99: baseLatency * 4 + Math.random() * 200,
      throughput: Math.floor(Math.random() * 1000) + 200,
      activeConnections: Math.floor(Math.random() * 300) + 50,
    },
    memory: {
      allocCount: Math.floor(Math.random() * 3000000) + 500000,
      allocBytes: Math.floor(Math.random() * 8589934592) + 1073741824,
      freeCount: Math.floor(Math.random() * 2900000) + 490000,
      leakSuspected: status === "degraded" && Math.random() > 0.7,
      heapSize: Math.floor(baseMemoryMB * 1024 * 1024), // Convert MB to bytes
      gcCount: Math.floor(Math.random() * 300) + 50,
      history: memoryHistory,
    },
    cpu: {
      usagePercent: baseCpuUsage + (Math.random() - 0.5) * 10,
      contextSwitches: Math.floor(Math.random() * 100000) + 10000,
      runqueueLatency: Math.random() * 5 + 0.5,
      history: cpuHistory,
    },
    network: {
      bytesReceived: Math.floor(Math.random() * 2147483648) + 268435456,
      bytesSent: Math.floor(Math.random() * 4294967296) + 536870912,
      connectionsActive: Math.floor(Math.random() * 200) + 50,
      connectionsTotal: Math.floor(Math.random() * 15000) + 5000,
      history: networkHistory,
    },
    filesystem: {
      readsCount: Math.floor(Math.random() * 30000) + 5000,
      writesCount: Math.floor(Math.random() * 20000) + 2000,
      readBytes: Math.floor(Math.random() * 2097152000) + 524288000,
      writeBytes: Math.floor(Math.random() * 1572864000) + 209715200,
      readLatencyAvg: Math.random() * 8 + 1,
      writeLatencyAvg: Math.random() * 15 + 3,
      history: filesystemHistory,
    },
    sloStatus: status,
    sloScore: baseScore + Math.random() * 10 - 5,
    violations:
      status === "violated"
        ? [
            "P95 latency exceeds target",
            "Error rate above threshold",
            "Throughput below target",
          ]
        : status === "degraded"
        ? ["P95 latency approaching limit"]
        : [],
    lastUpdated: Math.floor(Date.now() / 1000),
  };
};

// Generate 53 mock services across different namespaces and statuses
const mockServices: ServiceSLO[] = [
  // Production namespace (30 services)
  ...Array.from({ length: 15 }, (_, i) =>
    generateMockService(i, "production", "healthy")
  ),
  ...Array.from({ length: 10 }, (_, i) =>
    generateMockService(i + 15, "production", "degraded")
  ),
  ...Array.from({ length: 5 }, (_, i) =>
    generateMockService(i + 25, "production", "violated")
  ),

  // Staging namespace (13 services)
  ...Array.from({ length: 7 }, (_, i) =>
    generateMockService(i + 30, "staging", "healthy")
  ),
  ...Array.from({ length: 4 }, (_, i) =>
    generateMockService(i + 37, "staging", "degraded")
  ),
  ...Array.from({ length: 2 }, (_, i) =>
    generateMockService(i + 41, "staging", "violated")
  ),

  // Development namespace (10 services)
  ...Array.from({ length: 5 }, (_, i) =>
    generateMockService(i + 43, "development", "healthy")
  ),
  ...Array.from({ length: 3 }, (_, i) =>
    generateMockService(i + 48, "development", "degraded")
  ),
  ...Array.from({ length: 2 }, (_, i) =>
    generateMockService(i + 51, "development", "violated")
  ),
];

export default ClusterServices;
