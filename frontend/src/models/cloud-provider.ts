// Update the import path to the correct module location or ensure the module exists
import {
    ActivationStatus,
    ProviderConnection,
    CloudProvider as ProtoCloudProvider,
} from "@proto/backend/backend";

type CloudProviderConnection = ProviderConnection & Record<string, string | number | undefined>;

interface AddProviderModalProps {
    open: boolean;
    mode: "create" | "edit";
    onOpenChange: (open: boolean) => void;
    newProvider: CloudProviderConnection;
    setNewProvider: (provider: CloudProviderConnection) => void;
    onSave: () => void;
}

function EmptyCloudProvider(): CloudProviderConnection {
    return {
        createdAt: "",
        id: "-1",
        secretId: "",
        provider: ProtoCloudProvider.UNRECOGNIZED,
        status: ActivationStatus.UNRECOGNIZED,
    }
}

export { CloudProviderConnection, EmptyCloudProvider, AddProviderModalProps }