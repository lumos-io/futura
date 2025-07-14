interface CloudProvider {
    id?: number;
    name: string;
    account: string;
    roleName: string;
}

interface AddProviderModalProps {
    open: boolean;
    mode: "create" | "edit";
    onOpenChange: (open: boolean) => void;
    newProvider: CloudProvider;
    setNewProvider: (provider: CloudProvider) => void;
    onSave: () => void;
}

export { CloudProvider, AddProviderModalProps }