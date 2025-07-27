import { useEffect, useState } from "react";
import {
    ActivationStatus,
    CreateProviderConnectionRequest,
    SecretName,
} from "@proto/backend/backend";
import { useAuth } from "@/hooks/auth-provider";
import { CloudProviderConnection, ClusterInfo, EmptyCloudProvider } from "@/models/cloud-provider";

export type FetchClustersResultEvent = {
    providerConnectionId: number;
    organizationId: number;
    data: ClusterInfo[];
    error: string;
    status: string;
};

export const useCloudProviders = () => {
    const { user } = useAuth();
    const orgId = user?.organizationId;

    const [connectedProviders, setConnectedProviders] = useState<CloudProviderConnection[]>([]);
    const [dialogOpen, setDialogOpen] = useState(false);
    const [newProvider, setNewProvider] = useState<CloudProviderConnection>(EmptyCloudProvider);
    const [deleteTarget, setDeleteTarget] = useState<CloudProviderConnection | null>(null);

    useEffect(() => {
        const fetchProviders = async () => {
            const res = await fetch(`/api/organizations/${orgId}/connects`);
            const data = await res.json();
            setConnectedProviders(data.data);
        };

        if (orgId) fetchProviders();
    }, [orgId]);

    const handleSave = async (
        connectionName: string,
        credentials: Record<string, string>
    ) => {
        const input = CreateProviderConnectionRequest.fromJSON({
            provider: newProvider.provider,
            connection_name: connectionName,
            secret_name: SecretName.ACCESS_CREDENTIALS,
            credentials,
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
        if (!deleteTarget) { return; }

        await fetch(`/api/organizations/${orgId}/connects/${deleteTarget.id}`, {
            method: "DELETE",
        });

        setConnectedProviders((prev) =>
            prev.filter((p) => p.id !== deleteTarget.id)
        );
        setDeleteTarget(null);
    };

    const disableDeletionForConnection = (status: ActivationStatus) => {
        return status === ActivationStatus.IN_PROGRESS;
    };

    const openAdd = () => {
        setNewProvider(EmptyCloudProvider);
        setDialogOpen(true);
    };

    return {
        connectedProviders,
        setConnectedProviders,
        dialogOpen,
        setDialogOpen,
        newProvider,
        setNewProvider,
        deleteTarget,
        setDeleteTarget,
        handleSave,
        handleDelete,
        disableDeletionForConnection,
        openAdd,
        orgId,
    };
};
