// Update the import path to the correct module location or ensure the module exists
import {
    ActivationStatus,
    ProviderConnection,
    CloudProvider as ProtoCloudProvider,
} from "@proto/backend/backend";

export type CloudProviderConnection = ProviderConnection & Record<string, string | number | undefined>;

export interface AddProviderModalProps {
    open: boolean;
    mode: "create" | "edit";
    onOpenChange: (open: boolean) => void;
    newProvider: CloudProviderConnection;
    setNewProvider: (provider: CloudProviderConnection) => void;
    onSave: () => void;
}

export function EmptyCloudProvider(): CloudProviderConnection {
    return {
        created_at: "",
        id: -1,
        secret_id: "",
        connection_name: "",
        provider: ProtoCloudProvider.UNRECOGNIZED,
        status: ActivationStatus.UNRECOGNIZED,
        imported_clusters: 0,
    };
}

export type ClusterInfo = {
    name: string,
}