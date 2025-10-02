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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
} from "@/components/ui/sheet";
import { DeleteUserDialog } from "./components/delete-user-dialog";
import { InviteUserSheet } from "./components/invite-user-sheet";
import { Search, UserPlus, Mail, Calendar, Clock, Trash2 } from "lucide-react";

interface UsersProps {
  title: string;
}

type UserRole = "admin" | "developer" | "viewer" | "manager";

type User = {
  id: number;
  firstName: string;
  lastName: string;
  email: string;
  role: UserRole;
  teams: string[];
  createdAt: string; // ISO date string
  lastAccess: string; // ISO date string
  status: "active" | "inactive" | "invited";
};

const Users: React.FC<UsersProps> = ({ title }) => {
  const { user: currentUser } = useAuth();

  const [users, setUsers] = useState<User[]>([]);
  const [filteredUsers, setFilteredUsers] = useState<User[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage] = useState(10);

  // Edit user state
  const [editSheetOpen, setEditSheetOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [editForm, setEditForm] = useState({
    firstName: "",
    lastName: "",
    email: "",
    role: "" as UserRole,
    teams: [] as string[],
    status: "" as User["status"],
  });

  // Delete confirmation state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingUser, setDeletingUser] = useState<User | null>(null);
  const [saving, setSaving] = useState(false);
  const [dialogKey, setDialogKey] = useState(0);

  // Invite user state
  const [inviteSheetOpen, setInviteSheetOpen] = useState(false);

  const orgId = currentUser?.organizationId;

  // Available teams for selection
  const availableTeams = [
    "Engineering",
    "Product",
    "Design",
    "Marketing",
    "Sales",
    "Operations",
    "DevOps",
    "Security",
    "Data",
    "Support",
    "Finance",
    "HR",
  ];

  // Fetch users from API
  useEffect(() => {
    if (!orgId) return;

    const fetchUsers = async () => {
      try {
        const res = await fetch(`/api/organizations/${orgId}/users`);
        if (!res.ok) throw new Error("Failed to fetch users");

        const data = await res.json();
        setUsers(data.data || []);
        setFilteredUsers(data.data || []);
      } catch (err) {
        console.error("Error fetching users:", err);
        setUsers([]);
        setFilteredUsers([]);
      }
    };

    fetchUsers();
  }, [orgId]);

  // Cleanup effect for scroll locks when component unmounts
  useEffect(() => {
    return () => {
      // Cleanup on unmount
      document.body.style.pointerEvents = "";
      document.body.style.overflow = "";
      document.body.style.paddingRight = "";
      document.body.removeAttribute("data-scroll-locked");
      document.documentElement.removeAttribute("data-scroll-locked");
    };
  }, []);

  // Filter users
  useEffect(() => {
    let filtered = users;

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (u) =>
          u.firstName.toLowerCase().includes(query) ||
          u.lastName.toLowerCase().includes(query) ||
          u.email.toLowerCase().includes(query) ||
          u.teams.some((t) => t.toLowerCase().includes(query))
      );
    }

    if (roleFilter !== "all") {
      filtered = filtered.filter((u) => u.role === roleFilter);
    }

    if (statusFilter !== "all") {
      filtered = filtered.filter((u) => u.status === statusFilter);
    }

    setFilteredUsers(filtered);
    setCurrentPage(1); // Reset to first page when filters change
  }, [searchQuery, roleFilter, statusFilter, users]);

  // Pagination
  const totalPages = Math.ceil(filteredUsers.length / itemsPerPage);
  const startIndex = (currentPage - 1) * itemsPerPage;
  const endIndex = startIndex + itemsPerPage;
  const paginatedUsers = filteredUsers.slice(startIndex, endIndex);

  const getRoleBadge = (role: UserRole) => {
    const variants: Record<UserRole, { color: string; label: string }> = {
      admin: { color: "bg-red-500 text-white", label: "Admin" },
      manager: { color: "bg-blue-500 text-white", label: "Manager" },
      developer: { color: "bg-green-500 text-white", label: "Developer" },
      viewer: { color: "bg-gray-500 text-white", label: "Viewer" },
    };

    const variant = variants[role];
    return <Badge className={variant.color}>{variant.label}</Badge>;
  };

  const getStatusBadge = (status: User["status"]) => {
    switch (status) {
      case "active":
        return <Badge className="bg-green-500 text-white">Active</Badge>;
      case "inactive":
        return (
          <Badge variant="outline" className="text-gray-500">
            Inactive
          </Badge>
        );
      case "invited":
        return <Badge className="bg-yellow-500 text-white">Invited</Badge>;
    }
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  const formatLastAccess = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 60) {
      return `${diffMins}m ago`;
    } else if (diffHours < 24) {
      return `${diffHours}h ago`;
    } else if (diffDays < 7) {
      return `${diffDays}d ago`;
    } else {
      return formatDate(dateStr);
    }
  };

  const getUserStats = () => {
    return {
      total: filteredUsers.length,
      active: filteredUsers.filter((u) => u.status === "active").length,
      invited: filteredUsers.filter((u) => u.status === "invited").length,
      admins: filteredUsers.filter((u) => u.role === "admin").length,
    };
  };

  const stats = getUserStats();

  // Handle edit user
  const handleEditUser = (user: User) => {
    setEditingUser(user);
    setEditForm({
      firstName: user.firstName,
      lastName: user.lastName,
      email: user.email,
      role: user.role,
      teams: [...user.teams],
      status: user.status,
    });
    setEditSheetOpen(true);
  };

  // Handle save user
  const handleSaveUser = async () => {
    if (!editingUser || !orgId) return;

    setSaving(true);
    try {
      const res = await fetch(
        `/api/organizations/${orgId}/users/${editingUser.id}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(editForm),
        }
      );

      if (!res.ok) throw new Error("Failed to update user");

      const data = await res.json();
      const updatedUser = data.data;

      // Update local state
      setUsers((prev) =>
        prev.map((u) => (u.id === editingUser.id ? updatedUser : u))
      );

      setEditSheetOpen(false);
      setEditingUser(null);
    } catch (err) {
      console.error("Error updating user:", err);
      alert("Failed to update user. Please try again.");
    } finally {
      setSaving(false);
    }
  };

  // Handle delete user - Close sheet first, then open dialog
  const handleDeleteUser = (user: User) => {
    setDeletingUser(user);
    // Close the Sheet first to prevent scroll lock issues
    setEditSheetOpen(false);
    // Wait for Sheet to close before opening AlertDialog
    setTimeout(() => {
      setDeleteDialogOpen(true);
    }, 200);
  };

  // Force cleanup function - only reset styles, don't remove DOM nodes
  const forceCleanupScrollLock = () => {
    // Reset body styles
    document.body.style.pointerEvents = "";
    document.body.style.overflow = "";
    document.body.style.paddingRight = "";

    // Remove inert attributes
    document.body.removeAttribute("data-scroll-locked");
    document.documentElement.removeAttribute("data-scroll-locked");
  };

  // Confirm delete
  const confirmDelete = async () => {
    if (!deletingUser || !orgId) return;

    try {
      const res = await fetch(
        `/api/organizations/${orgId}/users/${deletingUser.id}`,
        {
          method: "DELETE",
        }
      );

      if (!res.ok) throw new Error("Failed to delete user");

      // Remove from local state
      setUsers((prev) => prev.filter((u) => u.id !== deletingUser.id));

      // Close dialog and clear state
      setDeleteDialogOpen(false);

      // Wait for dialog to fully close before cleanup
      setTimeout(() => {
        setDeletingUser(null);
        setEditingUser(null);
        // Increment key to force remount on next open
        setDialogKey((prev) => prev + 1);
        // Force cleanup
        forceCleanupScrollLock();
      }, 300);
    } catch (err) {
      console.error("Error deleting user:", err);
      setDeleteDialogOpen(false);
      setTimeout(() => {
        setDialogKey((prev) => prev + 1);
        forceCleanupScrollLock();
      }, 300);
      alert("Failed to delete user. Please try again.");
    }
  };

  // Toggle team selection
  const toggleTeam = (team: string) => {
    setEditForm((prev) => ({
      ...prev,
      teams: prev.teams.includes(team)
        ? prev.teams.filter((t) => t !== team)
        : [...prev.teams, team],
    }));
  };

  // Handle invite user
  const handleInviteUser = async (data: {
    firstName: string;
    lastName: string;
    email: string;
    role: UserRole;
    teams: string[];
  }) => {
    if (!orgId) return;

    setSaving(true);
    try {
      const res = await fetch(`/api/organizations/${orgId}/users/invite`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });

      if (!res.ok) throw new Error("Failed to invite user");

      const result = await res.json();
      const newUser = result.data;

      setUsers((prev) => [...prev, newUser]);
      setInviteSheetOpen(false);
    } catch (err) {
      console.error("Error inviting user:", err);
      alert("Failed to invite user. Please try again.");
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
            Manage user access, roles, and team memberships
          </p>
        </div>
        <button
          onClick={() => setInviteSheetOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
        >
          <UserPlus className="h-4 w-4" />
          Invite User
        </button>
      </div>

      {/* Stats Overview */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Users</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Active</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-500">
              {stats.active}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Invited</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-yellow-500">
              {stats.invited}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Admins</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-500">
              {stats.admins}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <div className="flex flex-col md:flex-row gap-4">
        <div className="flex-1">
          <div className="relative">
            <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search by name, email, or team..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>

        <div className="w-full md:w-40">
          <Select value={roleFilter} onValueChange={setRoleFilter}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Roles</SelectItem>
              <SelectItem value="admin">Admin</SelectItem>
              <SelectItem value="manager">Manager</SelectItem>
              <SelectItem value="developer">Developer</SelectItem>
              <SelectItem value="viewer">Viewer</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="w-full md:w-40">
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Status</SelectItem>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="inactive">Inactive</SelectItem>
              <SelectItem value="invited">Invited</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Users Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg">Users</CardTitle>
              <CardDescription>
                Showing {startIndex + 1}-
                {Math.min(endIndex, filteredUsers.length)} of{" "}
                {filteredUsers.length} user(s)
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {/* Table */}
          <div className="overflow-x-auto">
            <table className="w-full table-fixed">
              <colgroup>
                <col style={{ width: "18%" }} />
                <col style={{ width: "20%" }} />
                <col style={{ width: "10%" }} />
                <col style={{ width: "8%" }} />
                <col style={{ width: "16%" }} />
                <col style={{ width: "11%" }} />
                <col style={{ width: "11%" }} />
                <col style={{ width: "6%" }} />
              </colgroup>
              <thead>
                <tr className="border-b">
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    User
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Email
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Role
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Status
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Teams
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Created
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Last Access
                  </th>
                  <th className="text-left py-3 px-4 font-medium text-sm">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody>
                {paginatedUsers.map((user) => (
                  <tr
                    key={user.id}
                    className="border-b hover:bg-muted/50 transition-colors"
                  >
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-3">
                        <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center text-primary font-semibold">
                          {user.firstName[0]}
                          {user.lastName[0]}
                        </div>
                        <div>
                          <div className="font-medium">
                            {user.firstName} {user.lastName}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <Mail className="h-3 w-3" />
                        {user.email}
                      </div>
                    </td>
                    <td className="py-3 px-4">{getRoleBadge(user.role)}</td>
                    <td className="py-3 px-4">{getStatusBadge(user.status)}</td>
                    <td className="py-3 px-4">
                      <div className="flex flex-wrap gap-1">
                        {user.teams.slice(0, 2).map((team) => (
                          <Badge
                            key={team}
                            variant="outline"
                            className="text-xs"
                          >
                            {team}
                          </Badge>
                        ))}
                        {user.teams.length > 2 && (
                          <Badge variant="outline" className="text-xs">
                            +{user.teams.length - 2}
                          </Badge>
                        )}
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <Calendar className="h-3 w-3" />
                        {formatDate(user.createdAt)}
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <Clock className="h-3 w-3" />
                        {formatLastAccess(user.lastAccess)}
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <button
                        onClick={() => handleEditUser(user)}
                        className="text-sm text-primary hover:underline"
                      >
                        Edit
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {filteredUsers.length === 0 && (
              <div className="text-center py-12 text-muted-foreground">
                No users found matching your filters
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

                {/* Page numbers */}
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

      {/* Edit User Sheet */}
      <Sheet open={editSheetOpen} onOpenChange={setEditSheetOpen}>
        <SheetContent className="w-full sm:max-w-2xl overflow-y-auto p-0">
          {editingUser && (
            <div className="flex flex-col h-full">
              {/* Header */}
              <div className="px-6 py-5 border-b">
                <div className="flex items-center gap-4">
                  <div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center text-primary font-semibold text-lg">
                    {editingUser.firstName[0]}
                    {editingUser.lastName[0]}
                  </div>
                  <div>
                    <SheetTitle className="text-xl">
                      {editingUser.firstName} {editingUser.lastName}
                    </SheetTitle>
                    <SheetDescription className="text-sm">
                      Update user information and permissions
                    </SheetDescription>
                  </div>
                </div>
              </div>

              {/* Form Content */}
              <div className="flex-1 overflow-y-auto px-6 py-6">
                <div className="space-y-6">
                  {/* Personal Information Section */}
                  <div className="space-y-4">
                    <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                      Personal Information
                    </h3>
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <label className="text-sm font-medium">
                          First Name
                        </label>
                        <Input
                          value={editForm.firstName}
                          onChange={(e) =>
                            setEditForm({
                              ...editForm,
                              firstName: e.target.value,
                            })
                          }
                          placeholder="First name"
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="text-sm font-medium">Last Name</label>
                        <Input
                          value={editForm.lastName}
                          onChange={(e) =>
                            setEditForm({
                              ...editForm,
                              lastName: e.target.value,
                            })
                          }
                          placeholder="Last name"
                        />
                      </div>
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Email</label>
                      <Input
                        type="email"
                        value={editForm.email}
                        onChange={(e) =>
                          setEditForm({ ...editForm, email: e.target.value })
                        }
                        placeholder="email@company.com"
                      />
                    </div>
                  </div>

                  {/* Access & Permissions Section */}
                  <div className="space-y-4">
                    <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                      Access & Permissions
                    </h3>
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <label className="text-sm font-medium">Role</label>
                        <Select
                          value={editForm.role}
                          onValueChange={(value) =>
                            setEditForm({
                              ...editForm,
                              role: value as UserRole,
                            })
                          }
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="admin">
                              <div className="flex items-center gap-2">
                                <div className="h-2 w-2 rounded-full bg-red-500" />
                                Admin
                              </div>
                            </SelectItem>
                            <SelectItem value="manager">
                              <div className="flex items-center gap-2">
                                <div className="h-2 w-2 rounded-full bg-blue-500" />
                                Manager
                              </div>
                            </SelectItem>
                            <SelectItem value="developer">
                              <div className="flex items-center gap-2">
                                <div className="h-2 w-2 rounded-full bg-green-500" />
                                Developer
                              </div>
                            </SelectItem>
                            <SelectItem value="viewer">
                              <div className="flex items-center gap-2">
                                <div className="h-2 w-2 rounded-full bg-gray-500" />
                                Viewer
                              </div>
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="space-y-2">
                        <label className="text-sm font-medium">Status</label>
                        <Select
                          value={editForm.status}
                          onValueChange={(value) =>
                            setEditForm({
                              ...editForm,
                              status: value as User["status"],
                            })
                          }
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="active">
                              <div className="flex items-center gap-2">
                                <div className="h-2 w-2 rounded-full bg-green-500" />
                                Active
                              </div>
                            </SelectItem>
                            <SelectItem value="inactive">
                              <div className="flex items-center gap-2">
                                <div className="h-2 w-2 rounded-full bg-gray-400" />
                                Inactive
                              </div>
                            </SelectItem>
                            <SelectItem value="invited">
                              <div className="flex items-center gap-2">
                                <div className="h-2 w-2 rounded-full bg-yellow-500" />
                                Invited
                              </div>
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                  </div>

                  {/* Team Membership Section */}
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
                        Team Membership
                      </h3>
                      <span className="text-xs text-muted-foreground">
                        {editForm.teams.length} of {availableTeams.length}{" "}
                        selected
                      </span>
                    </div>
                    <div className="border rounded-lg divide-y max-h-64 overflow-y-auto">
                      {availableTeams.map((team) => (
                        <label
                          key={team}
                          className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-muted/50 transition-colors"
                        >
                          <input
                            type="checkbox"
                            checked={editForm.teams.includes(team)}
                            onChange={() => toggleTeam(team)}
                            className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                          />
                          <span className="text-sm font-medium">{team}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                </div>
              </div>

              {/* Footer */}
              <div className="px-6 py-4 border-t bg-muted/20">
                <div className="flex items-center justify-between gap-3">
                  <button
                    onClick={() => handleDeleteUser(editingUser)}
                    disabled={saving}
                    className="flex items-center gap-2 px-4 py-2 text-sm border border-red-500 text-red-500 rounded-md hover:bg-red-50 dark:hover:bg-red-950 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Trash2 className="h-4 w-4" />
                    Remove User
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
                      onClick={handleSaveUser}
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
      <DeleteUserDialog
        key={dialogKey}
        open={deleteDialogOpen}
        onOpenChange={(open) => {
          setDeleteDialogOpen(open);
          if (!open) {
            setTimeout(() => {
              setDeletingUser(null);
              setDialogKey((prev) => prev + 1);
              forceCleanupScrollLock();
            }, 300);
          }
        }}
        userName={`${deletingUser?.firstName} ${deletingUser?.lastName}`}
        onConfirm={confirmDelete}
        onCancel={() => {
          setDeletingUser(null);
          setDeleteDialogOpen(false);
        }}
      />

      {/* Invite User Sheet */}
      <InviteUserSheet
        open={inviteSheetOpen}
        onOpenChange={setInviteSheetOpen}
        onInviteUser={handleInviteUser}
        saving={saving}
        availableTeams={availableTeams}
      />
    </div>
  );
};

export default Users;
