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
  AlertTriangle,
  CheckCircle,
  ChevronDown,
  ChevronRight,
  Clock,
  Filter,
  Info,
  Search,
  XCircle,
} from "lucide-react";

interface EventsProps {
  title: string;
}

type EventSeverity = "Normal" | "Warning" | "Error";

type KubernetesEvent = {
  objectKind: string;
  objectName: string;
  objectNamespace: string;
  objectUid: string;
  eventName: string;
  eventReason: string;
  eventMessage: string;
  eventAction: string;
  eventSeverityText: EventSeverity;
  eventCount: number;
  objectTimestamp: number;
  nodeName?: string;
};

// API response type (snake_case from backend)
type KubernetesEventAPI = {
  object_kind?: string;
  objectKind?: string;
  object_name?: string;
  objectName?: string;
  object_namespace?: string;
  objectNamespace?: string;
  object_uid?: string;
  objectUid?: string;
  event_name?: string;
  eventName?: string;
  event_reason?: string;
  eventReason?: string;
  event_message?: string;
  eventMessage?: string;
  event_action?: string;
  eventAction?: string;
  event_severity_text?: string;
  eventSeverityText?: string;
  event_count?: number;
  eventCount?: number;
  object_timestamp?: number;
  objectTimestamp?: number;
  node_name?: string;
  nodeName?: string;
};

type TimelineGroup = {
  date: string;
  events: KubernetesEvent[];
};

type EventGroup = {
  kind: string;
  namespace: string;
  name: string;
  events: KubernetesEvent[];
  latestTimestamp: number;
};

const ClusterEvents: React.FC<EventsProps> = ({ title }) => {
  const { user } = useAuth();

  const [providersConnection, setProvidersConnection] = useState<
    CloudProviderConnection[]
  >([]);
  const [selectedProvider, setSelectedProvider] =
    useState<CloudProviderConnection>();
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [selectedCluster, setSelectedCluster] = useState<Cluster>();
  const [events, setEvents] = useState<KubernetesEvent[]>([]);
  const [filteredEvents, setFilteredEvents] = useState<KubernetesEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [severityFilter, setSeverityFilter] = useState<string>("all");
  const [kindFilter, setKindFilter] = useState<string>("all");
  const [viewMode, setViewMode] = useState<"timeline" | "grouped">("timeline");
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());

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

  // Set initial provider
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

  // Fetch events
  useEffect(() => {
    if (!selectedCluster || !orgId) return;

    const fetchEvents = async () => {
      setLoading(true);

      try {
        const res = await fetch(
          `/api/organizations/${orgId}/clusters/${selectedCluster.id}/events`
        );
        if (!res.ok) throw new Error("Failed to fetch events");

        const data = await res.json();

        // Map the response to match our interface
        const mappedEvents: KubernetesEvent[] = (data.data || []).map(
          (e: KubernetesEventAPI) => ({
            objectKind: e.object_kind || e.objectKind || "",
            objectName: e.object_name || e.objectName || "",
            objectNamespace: e.object_namespace || e.objectNamespace || "",
            objectUid: e.object_uid || e.objectUid || "",
            eventName: e.event_name || e.eventName || "",
            eventReason: e.event_reason || e.eventReason || "",
            eventMessage: e.event_message || e.eventMessage || "",
            eventAction: e.event_action || e.eventAction || "",
            eventSeverityText: (e.event_severity_text ||
              e.eventSeverityText ||
              "Normal") as EventSeverity,
            eventCount: e.event_count || e.eventCount || 1,
            objectTimestamp:
              e.object_timestamp || e.objectTimestamp || Math.floor(Date.now() / 1000),
            nodeName: e.node_name || e.nodeName,
          })
        );

        setEvents(mappedEvents);
        setFilteredEvents(mappedEvents);
      } catch (err) {
        console.error("Error fetching events:", err);
        // Mock data for development
        setEvents(mockEvents);
        setFilteredEvents(mockEvents);
      } finally {
        setLoading(false);
      }
    };

    fetchEvents();
  }, [selectedCluster, orgId]);

  // Filter events
  useEffect(() => {
    let filtered = events;

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (e) =>
          e.objectName.toLowerCase().includes(query) ||
          e.eventReason.toLowerCase().includes(query) ||
          e.eventMessage.toLowerCase().includes(query) ||
          e.objectNamespace.toLowerCase().includes(query)
      );
    }

    // Severity filter
    if (severityFilter !== "all") {
      filtered = filtered.filter((e) => e.eventSeverityText === severityFilter);
    }

    // Kind filter
    if (kindFilter !== "all") {
      filtered = filtered.filter((e) => e.objectKind === kindFilter);
    }

    setFilteredEvents(filtered);
  }, [searchQuery, severityFilter, kindFilter, events]);

  const getUniqueKinds = () => {
    const kinds = new Set(events.map((e) => e.objectKind));
    return Array.from(kinds).sort();
  };

  const getSeverityIcon = (severity: EventSeverity) => {
    switch (severity) {
      case "Error":
        return <XCircle className="h-4 w-4 text-red-500" />;
      case "Warning":
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      default:
        return <CheckCircle className="h-4 w-4 text-green-500" />;
    }
  };

  const getSeverityBadge = (severity: EventSeverity) => {
    const variant =
      severity === "Error"
        ? "destructive"
        : severity === "Warning"
        ? "default"
        : "secondary";

    return <Badge variant={variant}>{severity}</Badge>;
  };

  const formatTimestamp = (timestamp: number) => {
    const date = new Date(timestamp * 1000);
    return date.toLocaleString();
  };

  const formatRelativeTime = (timestamp: number) => {
    const now = Date.now();
    const eventTime = timestamp * 1000;
    const diff = now - eventTime;

    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    return `${days}d ago`;
  };

  const groupEventsByTimeline = (): TimelineGroup[] => {
    const groups: { [key: string]: KubernetesEvent[] } = {};

    filteredEvents.forEach((event) => {
      const date = new Date(event.objectTimestamp * 1000).toLocaleDateString();
      if (!groups[date]) {
        groups[date] = [];
      }
      groups[date].push(event);
    });

    return Object.entries(groups)
      .map(([date, events]) => ({
        date,
        events: events.sort((a, b) => b.objectTimestamp - a.objectTimestamp),
      }))
      .sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());
  };

  const groupEventsByResource = (): EventGroup[] => {
    const groups: { [key: string]: EventGroup } = {};

    filteredEvents.forEach((event) => {
      const key = `${event.objectKind}/${event.objectNamespace}/${event.objectName}`;

      if (!groups[key]) {
        groups[key] = {
          kind: event.objectKind,
          namespace: event.objectNamespace,
          name: event.objectName,
          events: [],
          latestTimestamp: event.objectTimestamp,
        };
      }

      groups[key].events.push(event);
      if (event.objectTimestamp > groups[key].latestTimestamp) {
        groups[key].latestTimestamp = event.objectTimestamp;
      }
    });

    return Object.values(groups).sort(
      (a, b) => b.latestTimestamp - a.latestTimestamp
    );
  };

  const toggleGroup = (groupKey: string) => {
    const newExpanded = new Set(expandedGroups);
    if (newExpanded.has(groupKey)) {
      newExpanded.delete(groupKey);
    } else {
      newExpanded.add(groupKey);
    }
    setExpandedGroups(newExpanded);
  };

  const getEventStats = () => {
    const stats = {
      total: filteredEvents.length,
      errors: filteredEvents.filter((e) => e.eventSeverityText === "Error")
        .length,
      warnings: filteredEvents.filter((e) => e.eventSeverityText === "Warning")
        .length,
      normal: filteredEvents.filter((e) => e.eventSeverityText === "Normal")
        .length,
    };
    return stats;
  };

  const stats = getEventStats();

  return (
    <div className="p-10 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-semibold">{title}</h1>
          <p className="text-muted-foreground mt-1">
            Monitor and analyze cluster events in real-time
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
              placeholder="Search events..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>
      </div>

      {/* Event Stats */}
      {!loading && selectedCluster && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">
                Total Events
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.total}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <XCircle className="h-4 w-4 text-red-500" />
                Errors
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-red-500">
                {stats.errors}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-yellow-500" />
                Warnings
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-yellow-500">
                {stats.warnings}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <CheckCircle className="h-4 w-4 text-green-500" />
                Normal
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-500">
                {stats.normal}
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Filters and View Toggle */}
      {!loading && selectedCluster && (
        <Card>
          <CardContent className="pt-6">
            <div className="flex flex-col md:flex-row gap-4 items-start md:items-center justify-between">
              <div className="flex gap-4 flex-wrap">
                <div className="flex items-center gap-2">
                  <Filter className="h-4 w-4 text-muted-foreground" />
                  <Select
                    value={severityFilter}
                    onValueChange={setSeverityFilter}
                  >
                    <SelectTrigger className="w-32">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Severities</SelectItem>
                      <SelectItem value="Normal">Normal</SelectItem>
                      <SelectItem value="Warning">Warning</SelectItem>
                      <SelectItem value="Error">Error</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <Select value={kindFilter} onValueChange={setKindFilter}>
                  <SelectTrigger className="w-40">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Resources</SelectItem>
                    {getUniqueKinds().map((kind) => (
                      <SelectItem key={kind} value={kind}>
                        {kind}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="flex gap-2">
                <button
                  onClick={() => setViewMode("timeline")}
                  className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                    viewMode === "timeline"
                      ? "bg-primary text-primary-foreground"
                      : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
                  }`}
                >
                  Timeline View
                </button>
                <button
                  onClick={() => setViewMode("grouped")}
                  className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                    viewMode === "grouped"
                      ? "bg-primary text-primary-foreground"
                      : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
                  }`}
                >
                  Grouped View
                </button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Loading State */}
      {loading && (
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-4 w-48" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-20 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Timeline View */}
      {!loading && selectedCluster && viewMode === "timeline" && (
        <div className="space-y-6">
          {groupEventsByTimeline().map((group) => (
            <div key={group.date}>
              <div className="flex items-center gap-2 mb-4">
                <Clock className="h-4 w-4 text-muted-foreground" />
                <h2 className="text-lg font-semibold">{group.date}</h2>
                <Badge variant="secondary">{group.events.length} events</Badge>
              </div>

              <div className="space-y-3 pl-6 border-l-2 border-muted">
                {group.events.map((event, idx) => (
                  <Card key={`${event.objectUid}-${idx}`} className="ml-4">
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-2">
                          {getSeverityIcon(event.eventSeverityText)}
                          <div>
                            <CardTitle className="text-base">
                              {event.eventReason}
                            </CardTitle>
                            <CardDescription className="text-xs mt-1">
                              {event.objectKind}/{event.objectNamespace}/
                              {event.objectName}
                            </CardDescription>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          {getSeverityBadge(event.eventSeverityText)}
                          <span className="text-xs text-muted-foreground">
                            {formatRelativeTime(event.objectTimestamp)}
                          </span>
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent className="pt-0">
                      <p className="text-sm text-muted-foreground">
                        {event.eventMessage}
                      </p>
                      <div className="flex items-center gap-4 mt-3 text-xs text-muted-foreground">
                        <span>Action: {event.eventAction || "N/A"}</span>
                        {event.eventCount > 1 && (
                          <Badge variant="outline">
                            Count: {event.eventCount}
                          </Badge>
                        )}
                        {event.nodeName && <span>Node: {event.nodeName}</span>}
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>
          ))}

          {filteredEvents.length === 0 && (
            <Card>
              <CardContent className="py-12 text-center">
                <Info className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <p className="text-muted-foreground">
                  No events found matching your filters
                </p>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {/* Grouped View */}
      {!loading && selectedCluster && viewMode === "grouped" && (
        <div className="space-y-3">
          {groupEventsByResource().map((group) => {
            const groupKey = `${group.kind}/${group.namespace}/${group.name}`;
            const isExpanded = expandedGroups.has(groupKey);
            const errorCount = group.events.filter(
              (e) => e.eventSeverityText === "Error"
            ).length;
            const warningCount = group.events.filter(
              (e) => e.eventSeverityText === "Warning"
            ).length;

            return (
              <Card key={groupKey}>
                <CardHeader
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => toggleGroup(groupKey)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      {isExpanded ? (
                        <ChevronDown className="h-4 w-4" />
                      ) : (
                        <ChevronRight className="h-4 w-4" />
                      )}
                      <div>
                        <CardTitle className="text-base">
                          {group.kind} / {group.name}
                        </CardTitle>
                        <CardDescription className="text-xs mt-1">
                          {group.namespace} • {group.events.length} events
                        </CardDescription>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {errorCount > 0 && (
                        <Badge variant="destructive">{errorCount} errors</Badge>
                      )}
                      {warningCount > 0 && (
                        <Badge variant="default">{warningCount} warnings</Badge>
                      )}
                      <span className="text-xs text-muted-foreground">
                        {formatRelativeTime(group.latestTimestamp)}
                      </span>
                    </div>
                  </div>
                </CardHeader>

                {isExpanded && (
                  <CardContent className="pt-0 space-y-2">
                    {group.events
                      .sort((a, b) => b.objectTimestamp - a.objectTimestamp)
                      .map((event, idx) => (
                        <div
                          key={idx}
                          className="flex items-start gap-3 p-3 rounded-lg border bg-muted/30"
                        >
                          {getSeverityIcon(event.eventSeverityText)}
                          <div className="flex-1">
                            <div className="flex items-center justify-between mb-1">
                              <span className="font-medium text-sm">
                                {event.eventReason}
                              </span>
                              <span className="text-xs text-muted-foreground">
                                {formatTimestamp(event.objectTimestamp)}
                              </span>
                            </div>
                            <p className="text-sm text-muted-foreground">
                              {event.eventMessage}
                            </p>
                          </div>
                        </div>
                      ))}
                  </CardContent>
                )}
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
};

// Mock data for development
const mockEvents: KubernetesEvent[] = [
  {
    objectKind: "Pod",
    objectName: "frontend-deployment-7d4f8c9b5d-xyz12",
    objectNamespace: "production",
    objectUid: "abc-123",
    eventName: "pod-started",
    eventReason: "Started",
    eventMessage: "Started container successfully",
    eventAction: "Starting",
    eventSeverityText: "Normal",
    eventCount: 1,
    objectTimestamp: Math.floor(Date.now() / 1000) - 300,
    nodeName: "node-1",
  },
  {
    objectKind: "Deployment",
    objectName: "backend-api",
    objectNamespace: "production",
    objectUid: "def-456",
    eventName: "scaling",
    eventReason: "ScalingReplicaSet",
    eventMessage: "Scaled up replica set backend-api-6b7d8f9c to 5",
    eventAction: "Scaling",
    eventSeverityText: "Normal",
    eventCount: 1,
    objectTimestamp: Math.floor(Date.now() / 1000) - 1800,
  },
  {
    objectKind: "Pod",
    objectName: "database-0",
    objectNamespace: "data",
    objectUid: "ghi-789",
    eventName: "failed-scheduling",
    eventReason: "FailedScheduling",
    eventMessage: "0/3 nodes are available: insufficient memory",
    eventAction: "Scheduling",
    eventSeverityText: "Warning",
    eventCount: 5,
    objectTimestamp: Math.floor(Date.now() / 1000) - 3600,
    nodeName: "node-2",
  },
  {
    objectKind: "Pod",
    objectName: "cache-redis-master-0",
    objectNamespace: "cache",
    objectUid: "jkl-012",
    eventName: "image-pull-failed",
    eventReason: "Failed",
    eventMessage: "Failed to pull image: ImagePullBackOff",
    eventAction: "Pulling",
    eventSeverityText: "Error",
    eventCount: 3,
    objectTimestamp: Math.floor(Date.now() / 1000) - 7200,
  },
];

export default ClusterEvents;
