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
import { Loader2, CheckCircle, AlertTriangle } from "lucide-react";
import { AddProviderModalProps } from "@/models/cloud-provider";

const providerOptions = ["ALIBABA", "AWS", "GCP", "DIGITALOCEAN", "AZURE"];

const AddProviderModal: React.FC<AddProviderModalProps> = ({
  open,
  mode,
  onOpenChange,
  newProvider,
  setNewProvider,
  onSave,
}) => {
  const [testing, setTesting] = useState(false);
  const [testSuccess, setTestSuccess] = useState<boolean | null>(null);

  const handleTestConnection = async () => {
    setTesting(true);
    setTestSuccess(null);

    try {
      const res = await fetch("/api/test-cloud-connection", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(newProvider),
      });

      const result = await res.json();
      setTestSuccess(res.ok && result?.success);
    } catch (err) {
      console.error(err);
      setTestSuccess(false);
    } finally {
      setTesting(false);
    }
  };

  const isValid =
    newProvider.name && newProvider.account && newProvider.roleName;

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
              value={newProvider.name}
              onValueChange={(value) =>
                setNewProvider({ ...newProvider, name: value })
              }
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

          <div className="space-y-2">
            <Label>Account</Label>
            <Input
              placeholder="Account ID / Number"
              value={newProvider.account}
              onChange={(e) =>
                setNewProvider({ ...newProvider, account: e.target.value })
              }
            />
          </div>

          <div className="space-y-2">
            <Label>Role Name</Label>
            <Input
              placeholder="Role Name"
              value={newProvider.roleName}
              onChange={(e) =>
                setNewProvider({ ...newProvider, roleName: e.target.value })
              }
            />
          </div>

          <div className="flex items-center gap-3">
            <Button
              type="button"
              variant="outline"
              disabled={!isValid || testing}
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
            {testSuccess === false && (
              <div className="flex items-center text-red-600 text-sm gap-1">
                <AlertTriangle className="w-4 h-4" /> Failed to connect
              </div>
            )}
          </div>

          <Button onClick={onSave} disabled={!testSuccess} className="w-full">
            {mode === "edit" ? "Update Provider" : "Save Provider"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default AddProviderModal;
