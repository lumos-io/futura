import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { CloudProviderConnection } from "@/models/cloud-provider";
import {
  cloudProviderFromJSON,
  cloudProviderToJSON,
  CloudProvider,
} from "@proto/backend/backend";
import { AlertTriangle, CheckCircle, Loader2 } from "lucide-react";
import { useAuth } from "@/hooks/auth_provider";

interface AddProviderModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  newProvider: CloudProviderConnection;
  setNewProvider: (provider: CloudProviderConnection) => void;
  onSave: (
    connectionName: string,
    credentials: { [key: string]: string }
  ) => void;
}

const providerOptions = ["ALIBABA", "AWS", "GCP", "DIGITALOCEAN", "AZURE"];

const providerFormFields: Record<
  string,
  { key: string; label: string; placeholder?: string }[]
> = {
  AWS: [
    { key: "account", label: "Account ID", placeholder: "123456789012" },
    { key: "accessKey", label: "Access Key", placeholder: "" },
    { key: "secretAccessKey", label: "Secret Access ID", placeholder: "" },
    { key: "region", label: "Region", placeholder: "eu-east-1" },
  ],
  GCP: [
    { key: "projectId", label: "Project ID", placeholder: "my-gcp-project" },
    { key: "credentialsJson", label: "Credentials JSON", placeholder: "{...}" },
  ],
  AZURE: [
    { key: "tenantId", label: "Tenant ID", placeholder: "" },
    { key: "clientId", label: "Client ID", placeholder: "" },
    { key: "clientSecret", label: "Client Secret", placeholder: "" },
  ],
  DIGITALOCEAN: [
    { key: "accessToken", label: "Access Token", placeholder: "" },
  ],
  ALIBABA: [
    { key: "accessKeyId", label: "Access Key ID", placeholder: "" },
    { key: "accessSecret", label: "Access Secret", placeholder: "" },
  ],
};

const AddProviderModal: React.FC<AddProviderModalProps> = ({
  open,
  onOpenChange,
  newProvider,
  setNewProvider,
  onSave,
}) => {
  const { user } = useAuth();

  const fields =
    providerFormFields[
      typeof newProvider.provider === "string" ? newProvider.provider : ""
    ] ?? [];

  const [testing, setTesting] = useState(false);
  const [testSuccess, setTestSuccess] = useState<boolean | null>(null);
  const [credentials, setCredentials] = useState<{ [key: string]: string }>({});
  const [connectionName, setConnectionName] = useState<string>("");

  const handleTestConnection = async () => {
    setTesting(true);
    setTestSuccess(null);

    const res = await fetch(
      `/api/organizations/${user?.organizationId}/connects/test-connection`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(newProvider),
      }
    );

    const result = await res.json();
    setTestSuccess(res.ok && result?.success);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Cloud Provider</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Provider</Label>
            <Select
              value={
                newProvider.provider === CloudProvider.UNRECOGNIZED
                  ? ""
                  : cloudProviderToJSON(newProvider.provider)
              }
              onValueChange={(value: string) => {
                setNewProvider({
                  ...newProvider,
                  provider: cloudProviderFromJSON(value),
                });
                setCredentials({}); // reset credentials on provider change
              }}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                {providerOptions.map((provider) => (
                  <SelectItem key={provider} value={provider}>
                    {provider}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {fields.length > 0 ? (
            <>
              <div className="space-y-2">
                <Label>Name</Label>
                <Input
                  placeholder={"Friendly Name"}
                  value={connectionName}
                  onChange={(e) => setConnectionName(e.target.value)}
                />
              </div>
              {fields.map((field) => (
                <div key={field.key} className="space-y-2">
                  <Label>{field.label}</Label>
                  <Input
                    placeholder={field.placeholder || ""}
                    value={credentials[field.key] ?? ""}
                    onChange={(e) =>
                      setCredentials({
                        ...credentials,
                        [field.key]: e.target.value,
                      })
                    }
                  />
                </div>
              ))}
            </>
          ) : (
            <p className="text-sm text-muted-foreground">
              No specific fields defined for this provider.
            </p>
          )}

          <div className="flex items-center gap-3">
            <Button
              type="button"
              variant="outline"
              disabled={testing}
              onClick={handleTestConnection}
            >
              {testing ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Testing...
                </>
              ) : (
                "Test Connection"
              )}
            </Button>

            {testSuccess === true && (
              <div className="flex items-center text-green-600 text-sm gap-1">
                <CheckCircle className="w-4 h-4" /> Connection successful
              </div>
            )}
            {testSuccess === false && testSuccess !== null && (
              <div className="flex items-center text-red-600 text-sm gap-1">
                <AlertTriangle className="w-4 h-4" /> Failed to connect
              </div>
            )}
          </div>

          <Button
            onClick={() => onSave(connectionName, credentials)}
            disabled={testSuccess ? true : false}
            className="w-full"
          >
            Save Provider
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default AddProviderModal;
