import React, { useEffect, useState } from "react";
import { useAuth } from "@/hooks/auth-provider";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
} from "@/components/ui/sheet";
import { DeleteTeamDialog } from "./components/delete-team-dialog";
import { CreateTeamSheet } from "./components/create-team-sheet";
import { Search, Users as UsersIcon, Plus, Trash2 } from "lucide-react";
import type { Team, TeamMember } from "@proto/backend/team";

interface TeamsProps {
  title: string;
}

const Teams: React.FC<TeamsProps> = ({ title }) => {
  const { user: currentUser } = useAuth();

  const [teams, setTeams] = useState<Team[]>([]);
  const [filteredTeams, setFilteredTeams] = useState<Team[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage] = useState(10);

  // Edit team state
  const [editSheetOpen, setEditSheetOpen] = useState(false);
  const [editingTeam, setEditingTeam] = useState<Team | null>(null);
  const [editForm, setEditForm] = useState({
    name: "",
    description: "",
    members: [] as TeamMember[],
  });

  // Delete confirmation state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingTeam, setDeletingTeam] = useState<Team | null>(null);
  const [saving, setSaving] = useState(false);
  const [dialogKey, setDialogKey] = useState(0);

  // Create team state
  const [createSheetOpen, setCreateSheetOpen] = useState(false);

  const orgId = currentUser?.organizationId;

  // Force cleanup function
  const forceCleanupScrollLock = () => {
    document.body.style.pointerEvents = "";
    document.body.style.overflow = "";
    document.body.style.paddingRight = "";
    document.body.removeAttribute("data-scroll-locked");
    document.documentElement.removeAttribute("data-scroll-locked");
  };

  // Fetch teams from API
  useEffect(() => {
    if (!orgId) return;

    const fetchTeams = async () => {
      try {
        const res = await fetch(`/api/organizations/${orgId}/teams`);
        if (!res.ok) throw new Error("Failed to fetch teams");

        const data = await res.json();
        setTeams(data.data || []);
        setFilteredTeams(data.data || []);
      } catch (err) {
        console.error("Error fetching teams:", err);
        setTeams([]);
        setFilteredTeams([]);
      }
    };

    fetchTeams();
  }, [orgId]);

  // Cleanup effect for scroll locks
  useEffect(() => {
    return () => {
      document.body.style.pointerEvents = "";
      document.body.style.overflow = "";
      document.body.style.paddingRight = "";
      document.body.removeAttribute("data-scroll-locked");
      document.documentElement.removeAttribute("data-scroll-locked");
    };
  }, []);

  // Filter teams
  useEffect(() => {
    let filtered = teams;

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (t) =>
          t.name.toLowerCase().includes(query) ||
          t.description.toLowerCase().includes(query)
      );
    }

    setFilteredTeams(filtered);
    setCurrentPage(1);
  }, [searchQuery, teams]);

  // Pagination
  const totalPages = Math.ceil(filteredTeams.length / itemsPerPage);
  const startIndex = (currentPage - 1) * itemsPerPage;
  const endIndex = startIndex + itemsPerPage;
  const paginatedTeams = filteredTeams.slice(startIndex, endIndex);

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  const getTeamStats = () => {
    const totalMembers = teams.reduce((acc, t) => acc + t.member_count, 0);
    const avgMembers =
      teams.length > 0 ? Math.round(totalMembers / teams.length) : 0;
    return {
      total: teams.length,
      totalMembers,
      avgMembers,
      largest: teams.reduce(
        (max, t) => (t.member_count > max ? t.member_count : max),
        0
      ),
    };
  };

  const stats = getTeamStats();

  // Handle edit team
  const handleEditTeam = (team: Team) => {
    setEditingTeam(team);
    setEditForm({
      name: team.name,
      description: team.description,
      members: [...team.members],
    });
    setEditSheetOpen(true);
  };

  // Handle save team
  const handleSaveTeam = async () => {
    if (!editingTeam || !orgId) return;

    setSaving(true);
    try {
      const res = await fetch(
        `/api/organizations/${orgId}/teams/${editingTeam.id}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(editForm),
        }
      );

      if (!res.ok) throw new Error("Failed to update team");

      const data = await res.json();
      const updatedTeam = data.data;

      setTeams((prev) =>
        prev.map((t) => (t.id === editingTeam.id ? updatedTeam : t))
      );

      setEditSheetOpen(false);
      setEditingTeam(null);
    } catch (err) {
      console.error("Error updating team:", err);
      alert("Failed to update team. Please try again.");
    } finally {
      setSaving(false);
    }
  };

  // Handle delete team
  const handleDeleteTeam = (team: Team) => {
    setDeletingTeam(team);
    setEditSheetOpen(false);
    setTimeout(() => {
      setDeleteDialogOpen(true);
    }, 200);
  };

  // Confirm delete
  const confirmDelete = async () => {
    if (!deletingTeam || !orgId) return;

    try {
      const res = await fetch(
        `/api/organizations/${orgId}/teams/${deletingTeam.id}`,
        {
          method: "DELETE",
        }
      );

      if (!res.ok) throw new Error("Failed to delete team");

      setTeams((prev) => prev.filter((t) => t.id !== deletingTeam.id));
      setDeleteDialogOpen(false);

      setTimeout(() => {
        setDeletingTeam(null);
        setEditingTeam(null);
        setDialogKey((prev) => prev + 1);
        forceCleanupScrollLock();
      }, 300);
    } catch (err) {
      console.error("Error deleting team:", err);
      setDeleteDialogOpen(false);
      setTimeout(() => {
        setDialogKey((prev) => prev + 1);
        forceCleanupScrollLock();
      }, 300);
      alert("Failed to delete team. Please try again.");
    }
  };

  // Handle create team
  const handleCreateTeam = async (data: {
    name: string;
    description: string;
  }) => {
    if (!orgId) return;

    setSaving(true);
    try {
      const res = await fetch(`/api/organizations/${orgId}/teams`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });

      if (!res.ok) throw new Error("Failed to create team");

      const result = await res.json();
      const newTeam = result.data;

      setTeams((prev) => [...prev, newTeam]);
      setCreateSheetOpen(false);
    } catch (err) {
      console.error("Error creating team:", err);
      alert("Failed to create team. Please try again.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="p-10 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-semibold">{title}</h1>
          <p className="text-muted-foreground mt-1">
            Organize users into teams for better access management
          </p>
        </div>
        <button
          onClick={() => setCreateSheetOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          Create Team
        </button>
      </div>

      {/* Stats Overview */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Teams</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Total Members
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-blue-500">
              {stats.totalMembers}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Avg Members/Team
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-500">
              {stats.avgMembers}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Largest Team</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-purple-500">
              {stats.largest}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search */}
      <div className="flex flex-col md:flex-row gap-4">
        <div className="flex-1">
          <div className="relative">
            <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search teams by name or description..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>
      </div>

      {/* Teams Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg">Teams</CardTitle>
              <CardDescription>
                Showing {startIndex + 1}-
                {Math.min(endIndex, filteredTeams.length)} of{" "}
                {filteredTeams.length} team(s)
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full table-fixed">
              <colgroup>
                <col style={{ width: "25%" }} />
                <col style={{ width: "40%" }} />
                <col style={{ width: "15%" }} />
                <col style={{ width: "15%" }} />
                <col style={{ width: "5%" }} />
              </colgroup>
              <thead>
                <tr className="border-b">
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Team Name
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Description
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Members
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Created
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody>
                {paginatedTeams.map((team) => (
                  <tr
                    key={team.id}
                    className="border-b hover:bg-muted/50 transition-colors"
                  >
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-3">
                        <div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                          <UsersIcon className="h-4 w-4" />
                        </div>
                        <div className="font-medium">{team.name}</div>
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <div className="text-sm text-muted-foreground truncate">
                        {team.description}
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <Badge variant="outline" className="font-mono">
                        {team.member_count}
                      </Badge>
                    </td>
                    <td className="py-3 px-4 text-sm text-muted-foreground">
                      {formatDate(team.created_at)}
                    </td>
                    <td className="py-3 px-4">
                      <button
                        onClick={() => handleEditTeam(team)}
                        className="text-sm text-primary hover:underline"
                      >
                        Edit
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {filteredTeams.length === 0 && (
              <div className="text-center py-12 text-muted-foreground">
                No teams found matching your search
              </div>
            )}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t">
              <div className="text-sm text-muted-foreground">
                Page {currentPage} of {totalPages}
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() =>
                    setCurrentPage((prev) => Math.max(1, prev - 1))
                  }
                  disabled={currentPage === 1}
                  className="px-3 py-1 text-sm border rounded-md hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Previous
                </button>

                <div className="flex gap-1">
                  {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                    let pageNum;
                    if (totalPages <= 5) {
                      pageNum = i + 1;
                    } else if (currentPage <= 3) {
                      pageNum = i + 1;
                    } else if (currentPage >= totalPages - 2) {
                      pageNum = totalPages - 4 + i;
                    } else {
                      pageNum = currentPage - 2 + i;
                    }

                    return (
                      <button
                        key={pageNum}
                        onClick={() => setCurrentPage(pageNum)}
                        className={`px-3 py-1 text-sm border rounded-md hover:bg-muted ${
                          currentPage === pageNum
                            ? "bg-primary text-primary-foreground"
                            : ""
                        }`}
                      >
                        {pageNum}
                      </button>
                    );
                  })}
                </div>

                <button
                  onClick={() =>
                    setCurrentPage((prev) => Math.min(totalPages, prev + 1))
                  }
                  disabled={currentPage === totalPages}
                  className="px-3 py-1 text-sm border rounded-md hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Edit Team Sheet */}
      <Sheet open={editSheetOpen} onOpenChange={setEditSheetOpen}>
        <SheetContent className="w-full sm:max-w-2xl overflow-y-auto p-0">
          {editingTeam && (
            <div className="flex flex-col h-full">
              {/* Header */}
              <div className="px-6 py-5 border-b">
                <div className="flex items-center gap-4">
                  <div className="h-12 w-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                    <UsersIcon className="h-6 w-6" />
                  </div>
                  <div>
                    <SheetTitle className="text-xl">
                      {editingTeam.name}
                    </SheetTitle>
                    <SheetDescription className="text-sm">
                      Update team information and members
                    </SheetDescription>
                  </div>
                </div>
              </div>

              {/* Form Content */}
              <div className="flex-1 overflow-y-auto px-6 py-6">
                <div className="space-y-6">
                  {/* Team Information */}
                  <div className="space-y-4">
                    <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                      Team Information
                    </h3>
                    <div className="space-y-4">
                      <div className="space-y-2">
                        <label className="text-sm font-medium">Team Name</label>
                        <Input
                          value={editForm.name}
                          onChange={(e) =>
                            setEditForm({
                              ...editForm,
                              name: e.target.value,
                            })
                          }
                          placeholder="e.g., Engineering, Product, Design"
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="text-sm font-medium">
                          Description
                        </label>
                        <textarea
                          value={editForm.description}
                          onChange={(e) =>
                            setEditForm({
                              ...editForm,
                              description: e.target.value,
                            })
                          }
                          placeholder="Describe the team's purpose and responsibilities"
                          className="w-full min-h-[100px] px-3 py-2 text-sm rounded-md border border-input bg-background resize-none focus:outline-none focus:ring-2 focus:ring-ring"
                        />
                      </div>
                    </div>
                  </div>

                  {/* Team Members */}
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                        Team Members
                      </h3>
                      <span className="text-xs text-muted-foreground">
                        {editForm.members.length} member(s)
                      </span>
                    </div>
                    <div className="border rounded-lg divide-y max-h-64 overflow-y-auto">
                      {editForm.members.map((member) => (
                        <div
                          key={member.id}
                          className="flex items-center justify-between px-4 py-3"
                        >
                          <div>
                            <div className="text-sm font-medium">
                              {member.name}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {member.email} · {member.role}
                            </div>
                          </div>
                          <button
                            onClick={() =>
                              setEditForm({
                                ...editForm,
                                members: editForm.members.filter(
                                  (m) => m.id !== member.id
                                ),
                              })
                            }
                            className="text-xs text-red-500 hover:underline"
                          >
                            Remove
                          </button>
                        </div>
                      ))}
                      {editForm.members.length === 0 && (
                        <div className="px-4 py-8 text-center text-sm text-muted-foreground">
                          No members in this team
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              {/* Footer */}
              <div className="px-6 py-4 border-t bg-muted/20">
                <div className="flex items-center justify-between gap-3">
                  <button
                    onClick={() => handleDeleteTeam(editingTeam)}
                    disabled={saving}
                    className="flex items-center gap-2 px-4 py-2 text-sm border border-red-500 text-red-500 rounded-md hover:bg-red-50 dark:hover:bg-red-950 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Trash2 className="h-4 w-4" />
                    Delete Team
                  </button>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setEditSheetOpen(false)}
                      disabled={saving}
                      className="px-4 py-2 text-sm border rounded-md hover:bg-muted transition-colors disabled:opacity-50"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={handleSaveTeam}
                      disabled={saving}
                      className="px-6 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 font-medium"
                    >
                      {saving ? "Saving..." : "Save Changes"}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </SheetContent>
      </Sheet>

      {/* Delete Confirmation Dialog */}
      <DeleteTeamDialog
        key={dialogKey}
        open={deleteDialogOpen}
        onOpenChange={(open) => {
          setDeleteDialogOpen(open);
          if (!open) {
            setTimeout(() => {
              setDeletingTeam(null);
              setDialogKey((prev) => prev + 1);
              forceCleanupScrollLock();
            }, 300);
          }
        }}
        teamName={deletingTeam?.name || ""}
        memberCount={deletingTeam?.member_count || 0}
        onConfirm={confirmDelete}
        onCancel={() => {
          setDeletingTeam(null);
          setDeleteDialogOpen(false);
        }}
      />

      {/* Create Team Sheet */}
      <CreateTeamSheet
        open={createSheetOpen}
        onOpenChange={setCreateSheetOpen}
        onCreateTeam={handleCreateTeam}
        saving={saving}
      />
    </div>
  );
};

export default Teams;
