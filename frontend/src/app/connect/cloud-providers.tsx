import React, { useEffect, useState } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Trash2, Cloud, CloudSun, CloudRain, Zap } from "lucide-react";
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
import { useAuth } from "@/hooks/auth_provider";
import {
  CreateProviderConnectionRequest,
  SecretName,
  cloudProviderFromJSON,
  activationStatusFromJSON,
  CloudProvider,
} from "@proto/backend/backend";
import { Badge } from "@/components/ui/badge";

const CloudIcon = ({ provider }: { provider: string }) => {
  switch (provider) {
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

  const handleSave = async (credentials: { [key: string]: string }) => {
    const input = CreateProviderConnectionRequest.fromJSON({
      provider: newProvider.provider,
      secretName: SecretName.ACCESS_CREDENTIALS,
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
          onSave={(credentials) => {
            handleSave(credentials);
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
              <CardHeader className="flex justify-between items-start space-y-0 pb-2">
                <div>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <CloudIcon provider={provider.provider} />
                    {cloudProviderFromJSON(provider.provider)}
                  </CardTitle>
                  <CardDescription className="text-xs text-muted-foreground">
                    Connected account for{" "}
                    {cloudProviderFromJSON(provider.provider)}
                    <span className="text-muted-foreground">
                      <b>Secret Id</b>
                      {provider.secretId}
                    </span>
                  </CardDescription>
                </div>

                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setDeleteTarget(provider)}
                    >
                      <Trash2 className="w-4 h-4 text-red-600" />
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>
                        Are you sure you want to delete this provider?
                      </AlertDialogTitle>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel onClick={() => setDeleteTarget(null)}>
                        Cancel
                      </AlertDialogCancel>
                      <AlertDialogAction onClick={handleDelete}>
                        Delete
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </CardHeader>

              <CardContent className="text-sm text-gray-700 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Status</span>
                  <Badge
                    className={`text-xs h-5 px-2 rounded-full ${
                      provider.status === "ACTIVE"
                        ? "bg-green-500 text-white"
                        : provider.status === "FAILED"
                        ? "bg-red-500 text-white"
                        : "bg-yellow-400 text-black"
                    }`}
                  >
                    {activationStatusFromJSON(provider.status)}
                  </Badge>
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
