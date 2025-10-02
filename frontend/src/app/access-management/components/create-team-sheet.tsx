import React, { useState } from "react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
} from "@/components/ui/sheet";
import { Input } from "@/components/ui/input";
import { Users as UsersIcon } from "lucide-react";

interface CreateTeamSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreateTeam: (data: { name: string; description: string }) => void;
  saving: boolean;
}

export const CreateTeamSheet: React.FC<CreateTeamSheetProps> = ({
  open,
  onOpenChange,
  onCreateTeam,
  saving,
}) => {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const handleCreate = () => {
    if (!name.trim()) {
      alert("Please enter a team name");
      return;
    }
    onCreateTeam({ name, description });
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen && !saving) {
      // Reset form when closing
      setName("");
      setDescription("");
    }
    onOpenChange(newOpen);
  };

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent className="w-full sm:max-w-xl overflow-y-auto p-0">
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="px-6 py-5 border-b">
            <div className="flex items-center gap-4">
              <div className="h-12 w-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                <UsersIcon className="h-6 w-6" />
              </div>
              <div>
                <SheetTitle className="text-xl">Create New Team</SheetTitle>
                <SheetDescription className="text-sm">
                  Create a new team to organize your users
                </SheetDescription>
              </div>
            </div>
          </div>

          {/* Form Content */}
          <div className="flex-1 overflow-y-auto px-6 py-6">
            <div className="space-y-6">
              <div className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">
                    Team Name <span className="text-red-500">*</span>
                  </label>
                  <Input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g., Engineering, Product, Design"
                    disabled={saving}
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">Description</label>
                  <textarea
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    placeholder="Describe the team's purpose and responsibilities"
                    className="w-full min-h-[120px] px-3 py-2 text-sm rounded-md border border-input bg-background resize-none focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
                    disabled={saving}
                  />
                </div>
              </div>

              <div className="p-4 bg-muted/50 rounded-lg border">
                <p className="text-sm text-muted-foreground">
                  💡 <strong>Tip:</strong> After creating the team, you can add
                  members by editing the team or by assigning users to the team
                  from the Users page.
                </p>
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="px-6 py-4 border-t bg-muted/20">
            <div className="flex justify-end gap-2">
              <button
                onClick={() => handleOpenChange(false)}
                disabled={saving}
                className="px-4 py-2 text-sm border rounded-md hover:bg-muted transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={saving || !name.trim()}
                className="px-6 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 font-medium"
              >
                {saving ? "Creating..." : "Create Team"}
              </button>
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
};
