import React from "react";
import { Button } from "@/components/ui/button";
import AddProviderModal from "@/app/connect/add-provider-modal";
import { CloudProviderCard } from "./cloud-provider-card";
import {
  FetchClustersResultEvent,
  useCloudProviders,
} from "./use-cloud-providers";
import {
  ActivationStatus,
  activationStatusFromJSON,
} from "@proto/backend/backend";
import { useSSE } from "@/hooks/sse-handler";
import { toast } from "sonner";

const CloudProviders: React.FC = () => {
  const {
    connectedProviders,
    setConnectedProviders,
    dialogOpen,
    setDialogOpen,
    newProvider,
    setNewProvider,
    handleSave,
    handleDelete,
    setDeleteTarget,
    disableDeletionForConnection,
    openAdd,
    orgId,
  } = useCloudProviders();

  const sseUrl = React.useMemo(() => {
    return orgId ? `/api/organizations/${orgId}/connects/fetch` : null;
  }, [orgId]);

  const { latest } = useSSE<FetchClustersResultEvent>(sseUrl || "", {
    event: "fetch_clusters_result",
    onMessage: (msg) => {
      const provider = connectedProviders.find(
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
    onError: (err) => {
      console.error("SSE failed:", err);
    },
  });

  React.useEffect(() => {
    if (!latest) {
      return;
    }

    setConnectedProviders((prevProviders) => {
      return prevProviders.map((provider) => {
        if (String(provider.id) === String(latest.providerConnectionId)) {
          console.log(latest);
          const n = latest.data ? latest.data.length : 0;
          return {
            ...provider,
            imported_clusters: n,
            status:
              latest.status === "SUCCESS"
                ? ActivationStatus.ACTIVE
                : ActivationStatus.FAILED,
          };
        }
        return provider;
      }) as typeof prevProviders;
    });
  }, [latest]);

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
            <CloudProviderCard
              provider={provider}
              disableDelete={disableDeletionForConnection(
                activationStatusFromJSON(provider.status)
              )}
              handleDelete={() => {
                setDeleteTarget(provider);
                handleDelete();
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default CloudProviders;
