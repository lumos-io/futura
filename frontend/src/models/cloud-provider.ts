const ActivationStatus = {
    ActiveStatus: "ACTIVE",
    InProgressStatus: "IN_PROGRESS",
    PendingStatus: "PENDING",
    SuspendedStatus: "SUSPENDED",
    FailedStatus: "FAILED",
} as const;

type ActivationStatus = typeof ActivationStatus[keyof typeof ActivationStatus];

interface CloudProvider {
    id?: number;
    provider: string;
    
    status: ActivationStatus;
    [key: string]: unknown; // dynamic fields
}

interface AddProviderModalProps {
    open: boolean;
    mode: "create" | "edit";
    onOpenChange: (open: boolean) => void;
    newProvider: CloudProvider;
    setNewProvider: (provider: CloudProvider) => void;
    onSave: () => void;
}

export { ActivationStatus, CloudProvider, AddProviderModalProps }