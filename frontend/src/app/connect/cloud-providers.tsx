import React, { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Trash2, Pencil } from "lucide-react";
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
import { CloudProvider } from "@/models/cloud-provider";
import { useAuth } from "@/hooks/auth_provider";
import {
  CreateProviderConnectionRequest,
  ActivationStatus,
  SecretIdName,
  cloudProviderFromJSON,
  activationStatusFromJSON,
  CloudProvider as ProtoCloudProvider,
} from "../../../../proto/gen/backend/backend";

const CloudProviders: React.FC = () => {
  const { user } = useAuth();

  const [connectedProviders, setConnectedProviders] = useState<CloudProvider[]>(
    []
  );
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editMode, setEditMode] = useState<false | CloudProvider>(false);
  const [newProvider, setNewProvider] = useState<CloudProvider>({
    createdAt: "",
    id: "-1",
    provider: ProtoCloudProvider.UNRECOGNIZED,
    status: ActivationStatus.UNRECOGNIZED,
  });
  const [deleteTarget, setDeleteTarget] = useState<CloudProvider | null>(null);

  const extractCredentials = (
    provider: CloudProvider
  ): { [key: string]: string } => {
    const { ...rest } = provider;

    const credentials: { [key: string]: string } = {};

    for (const [key, value] of Object.entries(rest)) {
      if (typeof value === "string") {
        credentials[key] = value;
      }
    }

    return credentials;
  };

  const handleSave = async () => {
    if (editMode) {
      // Edit mode
      const input = CreateProviderConnectionRequest.fromJSON({
        provider: newProvider?.name,
        secretId: SecretIdName.ACCESS_CREDENTIALS,
        credentials: extractCredentials(newProvider),
      });

      await fetch(
        `/api/organizations/${user?.organizationId}/connects/${editMode.id}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(input),
        }
      );

      console.log("edit newProvider: " + JSON.stringify(newProvider));

      setConnectedProviders((prev) =>
        prev.map((p) => (p.id === editMode.id ? { ...p, ...newProvider } : p))
      );
    } else {
      // Create mode
      const input = CreateProviderConnectionRequest.fromJSON({
        provider: newProvider?.name,
        secretId: SecretIdName.ACCESS_CREDENTIALS,
        credentials: extractCredentials(newProvider),
      });

      const res = await fetch(
        `/api/organizations/${user?.organizationId}/connects`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(input),
        }
      );

      console.log("create newProvider: " + JSON.stringify(newProvider));

      const created = await res.json();
      console.log("created: " + JSON.stringify(created.data));

      setConnectedProviders((prev) => {
        return [...prev, created.data];
      });
    }

    setDialogOpen(false);
    setNewProvider({
      createdAt: "",
      id: "-1",
      provider: ProtoCloudProvider.UNRECOGNIZED,
      status: ActivationStatus.UNRECOGNIZED,
    });
    setEditMode(false);
  };

  const handleDelete = async () => {
    if (!deleteTarget) {
      return;
    }

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

  // Fetch connected providers from API
  useEffect(() => {
    const handleGetAll = async () => {
      const orgId = user?.organizationId;
      const res = await fetch(`/api/organizations/${orgId}/connects`);

      const data = await res.json();
      console.log(data.data);

      setConnectedProviders(data.data);
    };
    handleGetAll();
  }, [user?.organizationId]);

  const openEdit = (provider: CloudProvider) => {
    setNewProvider(provider);
    setEditMode(provider);
    setDialogOpen(true);
  };

  const openAdd = () => {
    setNewProvider({
      createdAt: "",
      id: "-1",
      provider: ProtoCloudProvider.UNRECOGNIZED,
      status: ActivationStatus.UNRECOGNIZED,
    });
    setEditMode(false);
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
          onOpenChange={(open) => {
            setDialogOpen(open);
            if (!open) {
              setEditMode(false);
            }
          }}
          newProvider={newProvider}
          setNewProvider={setNewProvider}
          onSave={handleSave}
          mode={editMode ? "edit" : "create"}
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
            <Card>
              <CardHeader className="flex justify-between items-start">
                <div>
                  <CardTitle>
                    {cloudProviderFromJSON(provider.provider)}
                  </CardTitle>
                  <p className="text-xs text-muted-foreground">
                    {cloudProviderFromJSON(provider.provider)}
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => openEdit(provider)}
                  >
                    <Pencil className="w-4 h-4" />
                  </Button>

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
              <CardContent className="text-sm text-gray-700 space-y-1">
                <p>
                  <strong>Status:</strong>{" "}
                  {activationStatusFromJSON(provider.status)}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};

export default CloudProviders;
