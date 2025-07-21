import React, { useEffect, useState } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { toast } from "sonner";
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
import { useMultiWsConnections } from "@/hooks/use-connect-updates";

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

  // Handler for WebSocket updates
  const handleUpdate = (update: CloudProviderConnection) => {
    setConnectedProviders((prev) =>
      prev.map((p) => {
        if (p.id === update.id) {
          if (p.status !== update.status) {
            if (update.status === "ACTIVE") {
              toast.success(`Connection ${p.connection_name} is now active.`);
            } else if (update.status === "FAILED") {
              toast.error(`Connection ${p.connection_name} failed.`);
            } else if (update.status === "SUSPENDED") {
              toast(`Connection ${p.connection_name} was suspended.`, {
                icon: "⚠️",
              });
            }
          }
          return { ...p, status: update.status };
        }
        return p;
      })
    );
  };

  // Use the hook, passing all provider IDs and update handler
  useMultiWsConnections(
    connectedProviders.map((p) => p.id),
    {
      onUpdate: handleUpdate,
      wsBaseUrl: import.meta.env.VITE_WS_URL,
    }
  );

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

    const res = await fetch(
      `/api/organizations/${user?.organizationId}/connects`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      }
    );

    const created = await res.json();
    setConnectedProviders((prev) => [...prev, created.data]);

    setDialogOpen(false);
    setNewProvider(EmptyCloudProvider);
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;

    await fetch(
      `/api/organizations/${user?.organizationId}/connects/${deleteTarget.id}`,
      {
        method: "DELETE",
      }
    );
    setConnectedProviders((prev) =>
      prev.filter((p) => p.id !== deleteTarget.id)
    );
    setDeleteTarget(null);
  };

  useEffect(() => {
    const handleGetAll = async () => {
      const orgId = user?.organizationId;
      const res = await fetch(`/api/organizations/${orgId}/connects`);
      const data = await res.json();

      setConnectedProviders(data.data);
    };
    handleGetAll();
  }, [user?.organizationId]);

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
