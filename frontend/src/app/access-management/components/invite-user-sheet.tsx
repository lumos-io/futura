import React, { useState } from "react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
} from "@/components/ui/sheet";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { UserPlus } from "lucide-react";
import { UserRole } from "@proto/backend/user";

interface InviteUserSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onInviteUser: (data: {
    firstName: string;
    lastName: string;
    email: string;
    role: UserRole;
    teamIds: number[];
  }) => void;
  saving: boolean;
  availableTeams: Array<{ id: number; name: string }>;
}

export const InviteUserSheet: React.FC<InviteUserSheetProps> = ({
  open,
  onOpenChange,
  onInviteUser,
  saving,
  availableTeams,
}) => {
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<UserRole>(UserRole.USER_DEVELOPER);
  const [selectedTeamIds, setSelectedTeamIds] = useState<number[]>([]);

  const handleInvite = () => {
    if (!firstName.trim() || !lastName.trim() || !email.trim()) {
      alert("Please fill in all required fields");
      return;
    }

    // Basic email validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      alert("Please enter a valid email address");
      return;
    }

    onInviteUser({
      firstName,
      lastName,
      email,
      role,
      teamIds: selectedTeamIds,
    });
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen && !saving) {
      // Reset form when closing
      setFirstName("");
      setLastName("");
      setEmail("");
      setRole(UserRole.USER_DEVELOPER);
      setSelectedTeamIds([]);
    }
    onOpenChange(newOpen);
  };

  const toggleTeam = (teamId: number) => {
    setSelectedTeamIds((prev) =>
      prev.includes(teamId) ? prev.filter((id) => id !== teamId) : [...prev, teamId]
    );
  };

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent className="w-full sm:max-w-xl overflow-y-auto p-0">
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="px-6 py-5 border-b">
            <div className="flex items-center gap-4">
              <div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center text-primary">
                <UserPlus className="h-6 w-6" />
              </div>
              <div>
                <SheetTitle className="text-xl">Invite User</SheetTitle>
                <SheetDescription className="text-sm">
                  Send an invitation to a new user
                </SheetDescription>
              </div>
            </div>
          </div>

          {/* Form Content */}
          <div className="flex-1 overflow-y-auto px-6 py-6">
            <div className="space-y-6">
              {/* Personal Information */}
              <div className="space-y-4">
                <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                  Personal Information
                </h3>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">
                      First Name <span className="text-red-500">*</span>
                    </label>
                    <Input
                      value={firstName}
                      onChange={(e) => setFirstName(e.target.value)}
                      placeholder="John"
                      disabled={saving}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">
                      Last Name <span className="text-red-500">*</span>
                    </label>
                    <Input
                      value={lastName}
                      onChange={(e) => setLastName(e.target.value)}
                      placeholder="Doe"
                      disabled={saving}
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">
                    Email <span className="text-red-500">*</span>
                  </label>
                  <Input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="john.doe@company.com"
                    disabled={saving}
                  />
                </div>
              </div>

              {/* Access & Permissions */}
              <div className="space-y-4">
                <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                  Access & Permissions
                </h3>
                <div className="space-y-2">
                  <label className="text-sm font-medium">Role</label>
                  <Select
                    value={role}
                    onValueChange={(value) => setRole(value as UserRole)}
                    disabled={saving}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={UserRole.USER_ADMIN}>
                        <div className="flex items-center gap-2">
                          <div className="h-2 w-2 rounded-full bg-red-500" />
                          Admin
                        </div>
                      </SelectItem>
                      <SelectItem value={UserRole.USER_MANAGER}>
                        <div className="flex items-center gap-2">
                          <div className="h-2 w-2 rounded-full bg-blue-500" />
                          Manager
                        </div>
                      </SelectItem>
                      <SelectItem value={UserRole.USER_DEVELOPER}>
                        <div className="flex items-center gap-2">
                          <div className="h-2 w-2 rounded-full bg-green-500" />
                          Developer
                        </div>
                      </SelectItem>
                      <SelectItem value={UserRole.USER_VIEWER}>
                        <div className="flex items-center gap-2">
                          <div className="h-2 w-2 rounded-full bg-gray-500" />
                          Viewer
                        </div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Team Membership */}
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                    Team Membership
                  </h3>
                  <span className="text-xs text-muted-foreground">
                    {selectedTeamIds.length} of {availableTeams.length} selected
                  </span>
                </div>
                <div className="border rounded-lg divide-y max-h-64 overflow-y-auto">
                  {availableTeams.map((team) => (
                    <label
                      key={team.id}
                      className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-muted/50 transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={selectedTeamIds.includes(team.id)}
                        onChange={() => toggleTeam(team.id)}
                        disabled={saving}
                        className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary disabled:opacity-50"
                      />
                      <span className="text-sm font-medium">{team.name}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className="p-4 bg-muted/50 rounded-lg border">
                <p className="text-sm text-muted-foreground">
                  📧 <strong>Note:</strong> An invitation email will be sent to
                  the user with instructions to set up their account.
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
                onClick={handleInvite}
                disabled={
                  saving ||
                  !firstName.trim() ||
                  !lastName.trim() ||
                  !email.trim()
                }
                className="px-6 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 font-medium"
              >
                {saving ? "Sending Invite..." : "Send Invitation"}
              </button>
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
};
