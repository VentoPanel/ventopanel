"use client";

import { useState } from "react";
import { Users, Trash2, ShieldCheck, UserIcon } from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchUsers,
  updateUserRole,
  deleteUser,
  type User,
} from "@/lib/api";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { formatDistanceToNow } from "date-fns";

const ROLES = ["admin", "editor", "viewer"] as const;

const ROLE_COLORS: Record<string, string> = {
  admin: "bg-purple-100 text-purple-800",
  editor: "bg-blue-100 text-blue-800",
  viewer: "bg-gray-100 text-gray-700",
};

function RoleBadge({ role }: { role: string }) {
  return (
    <Badge
      variant="outline"
      className={cn("capitalize", ROLE_COLORS[role] ?? "bg-gray-100 text-gray-700")}
    >
      {role === "admin" && <ShieldCheck className="mr-1 h-3 w-3" />}
      {role !== "admin" && <UserIcon className="mr-1 h-3 w-3" />}
      {role}
    </Badge>
  );
}

export default function UsersPage() {
  const qc = useQueryClient();

  const { data: users = [], isLoading } = useQuery({
    queryKey: ["users"],
    queryFn: fetchUsers,
  });

  const rolesMutation = useMutation({
    mutationFn: ({ id, role }: { id: string; role: string }) =>
      updateUserRole(id, role),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      toast.success("Role updated");
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "Failed"),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteUser,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      toast.success("User deleted");
      setConfirm(null);
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed");
      setConfirm(null);
    },
  });

  const [confirm, setConfirm] = useState<User | null>(null);

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center gap-3">
        <Users className="h-6 w-6 text-muted-foreground" />
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Users</h2>
          <p className="text-muted-foreground">
            Manage team members and their roles
          </p>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium">
            {users.length} {users.length === 1 ? "member" : "members"}
          </CardTitle>
          <CardDescription>
            Roles: <strong>admin</strong> — full access ·{" "}
            <strong>editor</strong> — create & edit ·{" "}
            <strong>viewer</strong> — read-only
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading && (
            <p className="px-4 py-6 text-sm text-muted-foreground">
              Loading…
            </p>
          )}
          {!isLoading && users.length === 0 && (
            <p className="px-4 py-6 text-sm text-muted-foreground">
              No users found.
            </p>
          )}
          <ul className="divide-y">
            {users.map((user) => (
              <li
                key={user.id}
                className="flex flex-wrap items-center gap-3 px-4 py-3"
              >
                <div className="flex-1 min-w-0">
                  <p className="truncate font-medium text-sm">{user.email}</p>
                  <p className="text-xs text-muted-foreground">
                    Joined{" "}
                    {formatDistanceToNow(new Date(user.created_at), {
                      addSuffix: true,
                    })}
                  </p>
                </div>

                <RoleBadge role={user.role} />

                {/* Role selector */}
                <select
                  className="rounded border border-input bg-background px-2 py-1 text-xs text-foreground"
                  value={user.role}
                  disabled={rolesMutation.isPending}
                  onChange={(e) =>
                    rolesMutation.mutate({ id: user.id, role: e.target.value })
                  }
                >
                  {ROLES.map((r) => (
                    <option key={r} value={r}>
                      {r}
                    </option>
                  ))}
                </select>

                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive hover:text-destructive h-7 w-7 shrink-0"
                  onClick={() => setConfirm(user)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={!!confirm}
        title={`Delete "${confirm?.email}"?`}
        description="The user will lose access immediately. This cannot be undone."
        loading={deleteMutation.isPending}
        onConfirm={() => confirm && deleteMutation.mutate(confirm.id)}
        onCancel={() => setConfirm(null)}
      />
    </div>
  );
}
