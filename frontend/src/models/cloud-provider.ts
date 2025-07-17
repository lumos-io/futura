import { ProviderConnection } from "../../../proto/gen/backend/backend";

type CloudProvider = ProviderConnection & Record<string, string | number | undefined>;

interface AddProviderModalProps {
    open: boolean;
    mode: "create" | "edit";
    onOpenChange: (open: boolean) => void;
    newProvider: CloudProvider;
    setNewProvider: (provider: CloudProvider) => void;
    onSave: () => void;
}

export { CloudProvider, AddProviderModalProps }