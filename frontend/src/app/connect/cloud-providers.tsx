import React, { useEffect, useState } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Trash2, Cloud, CloudSun, CloudRain, Zap, Ghost } from "lucide-react";
import AddProviderModal from "@/app/connect/add-provider-modal";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import {
  CloudProviderConnection,
  EmptyCloudProvider,
} from "@/models/cloud-provider";
import { useAuth } from "@/hooks/auth-provider";
import {
  CreateProviderConnectionRequest,
  SecretName,
  CloudProvider,
  cloudProviderFromJSON,
  activationStatusFromJSON,
  ActivationStatus,
} from "@proto/backend/backend";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent } from "@/components/ui/tooltip";
import { TooltipTrigger } from "@radix-ui/react-tooltip";
import { useSSE } from "@/hooks/sse-handler";
import { toast } from "sonner";

const CloudIcon = ({ provider }: { provider: CloudProvider }) => {
  switch (cloudProviderFromJSON(provider)) {
    case CloudProvider.AWS:
      return <Cloud className="w-5 h-5 text-yellow-500" />;
    case CloudProvider.GCP:
      return <CloudSun className="w-5 h-5 text-blue-500" />;
    case CloudProvider.AZURE:
      return <Zap className="w-5 h-5 text-blue-700" />;
    case CloudProvider.DIGITALOCEAN:
      return <Cloud className="w-5 h-5 text-indigo-500" />;
    case CloudProvider.ALIBABA:
      return <CloudRain className="w-5 h-5 text-orange-500" />;
    case CloudProvider.KIND:
      return <Ghost className="w-5 h-5 text-purple-500" />;
    default:
      return null;
  }
};

const RenderActivationStatus = ({ status }: { status: ActivationStatus }) => {
  const s = activationStatusFromJSON(status);
  switch (s) {
    case ActivationStatus.ACTIVE:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-green-500 text-white`}
        >
          {" "}
          {s}
        </Badge>
      );
    case ActivationStatus.FAILED:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-red-500 text-white`}
        >
          {" "}
          {s}
        </Badge>
      );
    case ActivationStatus.IN_PROGRESS:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-blue-500 text-white`}
        >
          {" "}
          {s}
        </Badge>
      );
    case ActivationStatus.PENDING:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-gray-500 text-white`}
        >
          {" "}
          {s}
        </Badge>
      );
    case ActivationStatus.SUSPENDED:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-orange-500 text-white`}
        >
          {" "}
          {s}
        </Badge>
      );
    default:
      return null;
  }
};

type FetchClustersResultEvent = {
  providerConnectionId: number;
  organizationId: number;
  error: string;
  status: string;
};

const CloudProviders: React.FC = () => {
  const { user } = useAuth();

  const [connectedProviders, setConnectedProviders] = useState<
    CloudProviderConnection[]
  >([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newProvider, setNewProvider] =
    useState<CloudProviderConnection>(EmptyCloudProvider);
  const [deleteTarget, setDeleteTarget] =
    useState<CloudProviderConnection | null>(null);

  const orgId = user?.organizationId;

  const sseUrl = React.useMemo(() => {
    return orgId ? `/api/organizations/${orgId}/connects/result` : null;
  }, [orgId]);

  const connectedProvidersRef =
    React.useRef<CloudProviderConnection[]>(connectedProviders);
  // Keep the ref updated whenever connectedProviders changes
  useEffect(() => {
    connectedProvidersRef.current = connectedProviders;
  }, [connectedProviders]);

  const options = React.useMemo(
    () => ({
      event: "fetch_clusters_result",
      onMessage: (msg: FetchClustersResultEvent) => {
        const provider = connectedProvidersRef.current.find(
          (p) => String(p.id) === String(msg.providerConnectionId)
        );

        if (provider) {
          toast(
            `Provider "${provider.connection_name}" updated with status "${msg.status}"`
          );
        } else {
          toast(
            `Provider with ID ${msg.providerConnectionId} updated with status "${msg.status}"`
          );
        }
      },
      onError: (err: unknown) => {
        console.error("SSE failed:", err);
      },
    }),
    []
  );

  const { latest } = useSSE<FetchClustersResultEvent>(
    sseUrl ? sseUrl : "",
    options
  );

  useEffect(() => {
    if (latest) {
      setConnectedProviders((prevProviders) => {
        console.log("latest is updated: " + JSON.stringify(latest));
        return prevProviders.map((provider) => {
          console.log("current provider: " + JSON.stringify(provider));
          if (provider.id == String(latest.providerConnectionId)) {
            if (latest.status == "SUCCESS") {
              return {
                ...provider,
                status: ActivationStatus.ACTIVE,
              };
            } else {
              return {
                ...provider,
                status: ActivationStatus.FAILED,
              };
            }
          }
          return provider;
        });
      });
    }
  }, [latest]);

  const handleSave = async (
    connectionName: string,
    credentials: { [key: string]: string }
  ) => {
    const input = CreateProviderConnectionRequest.fromJSON({
      provider: newProvider.provider,
      connection_name: connectionName,
      secret_name: SecretName.ACCESS_CREDENTIALS,
      credentials: credentials,
    });

    const res = await fetch(`/api/organizations/${orgId}/connects`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });

    const created = await res.json();
    setConnectedProviders((prev) => [...prev, created.data]);

    setDialogOpen(false);
    setNewProvider(EmptyCloudProvider);
  };

  const handleDelete = async () => {
    if (!deleteTarget) {
      return;
    }

    await fetch(`/api/organizations/${orgId}/connects/${deleteTarget.id}`, {
      method: "DELETE",
    });
    setConnectedProviders((prev) =>
      prev.filter((p) => p.id !== deleteTarget.id)
    );
    setDeleteTarget(null);
  };

  useEffect(() => {
    const handleGetAll = async () => {
      const res = await fetch(`/api/organizations/${orgId}/connects`);
      const data = await res.json();

      setConnectedProviders(data.data);
    };
    handleGetAll();
  }, [orgId]);

  const disableDeletionForConnection = (status: ActivationStatus): boolean => {
    return activationStatusFromJSON(status) === ActivationStatus.IN_PROGRESS;
  };

  const openAdd = () => {
    setNewProvider(EmptyCloudProvider);
    setDialogOpen(true);
  };

  return (
    <div className="p-10">
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-3xl font-semibold text-gray-800">
          Cloud Providers
        </h1>
        <AddProviderModal
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          newProvider={newProvider}
          setNewProvider={setNewProvider}
          onSave={(connectionName, credentials) => {
            handleSave(connectionName, credentials);
          }}
        />
        <Button onClick={openAdd}>Add Provider</Button>
      </div>

      {connectedProviders.length === 0 ? (
        <p className="text-gray-500 text-center">
          No cloud providers connected.
        </p>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {connectedProviders.map((provider) => (
            <Card
              key={provider.id}
              className="shadow-md hover:shadow-lg transition-shadow rounded-2xl border border-muted"
            >
              <CardHeader className="pb-2 space-y-2">
                <div className="flex justify-between items-start">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <CloudIcon provider={provider.provider} />
                      <CardTitle className="text-lg">
                        {cloudProviderFromJSON(provider.provider)}
                      </CardTitle>
                    </div>
                    <CardDescription className="text-sm text-muted-foreground">
                      <div>{provider.connection_name}</div>
                      <div>{provider.secret_id}</div>
                    </CardDescription>
                  </div>

                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      {disableDeletionForConnection(provider.status) ? (
                        <Tooltip>
                          <TooltipTrigger className="cursor-not-allowed">
                            <Button
                              variant="ghost"
                              size="icon"
                              disabled
                              onClick={() => setDeleteTarget(provider)}
                            >
                              <Trash2 className="w-4 h-4 text-red-600" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>Fetching clusters metadata in progress...</p>
                          </TooltipContent>
                        </Tooltip>
                      ) : (
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => setDeleteTarget(provider)}
                        >
                          <Trash2 className="w-4 h-4 text-red-600" />
                        </Button>
                      )}
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>
                          Are you sure you want to delete this provider?
                        </AlertDialogTitle>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel
                          onClick={() => setDeleteTarget(null)}
                        >
                          Cancel
                        </AlertDialogCancel>
                        <AlertDialogAction onClick={handleDelete}>
                          Delete
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </CardHeader>

              <CardContent className="text-sm text-gray-700 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">
                    <b>Status</b>
                  </span>
                  <RenderActivationStatus status={provider.status} />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};

export default CloudProviders;
