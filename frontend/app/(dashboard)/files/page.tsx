"use client";

import {
  useState,
  useRef,
  useCallback,
  useEffect,
} from "react";
import {
  Folder,
  File,
  FileText,
  FileCode,
  FileImage,
  ArrowLeft,
  Upload,
  FolderPlus,
  MoreVertical,
  Pencil,
  Trash2,
  Download,
  Save,
  X,
  Loader2,
  HardDrive,
  ChevronRight,
} from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fmListDir,
  fmReadFile,
  fmWriteFile,
  fmDelete,
  fmCreateDir,
  fmRename,
  fmUpload,
  fmDownloadUrl,
  type FileItem,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { toast } from "sonner";
import { formatDistanceToNow } from "date-fns";
import { cn } from "@/lib/utils";

// ─── File icon by extension ───────────────────────────────────────────────────

const TEXT_EXTS = new Set([".txt", ".md", ".log", ".csv", ".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".env"]);
const CODE_EXTS = new Set([".js", ".ts", ".tsx", ".jsx", ".go", ".py", ".php", ".rb", ".rs", ".sh", ".bash", ".css", ".html", ".htm", ".sql"]);
const IMG_EXTS  = new Set([".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico"]);

function FileIcon({ item, className }: { item: FileItem; className?: string }) {
  const cls = cn("shrink-0", className);
  if (item.is_dir) return <Folder className={cn(cls, "text-amber-400")} />;
  if (CODE_EXTS.has(item.ext))  return <FileCode  className={cn(cls, "text-blue-400")} />;
  if (TEXT_EXTS.has(item.ext))  return <FileText  className={cn(cls, "text-gray-400")} />;
  if (IMG_EXTS.has(item.ext))   return <FileImage className={cn(cls, "text-pink-400")} />;
  return <File className={cn(cls, "text-gray-400")} />;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

// ─── Breadcrumbs ─────────────────────────────────────────────────────────────

function Breadcrumbs({ path, onNavigate }: { path: string; onNavigate: (p: string) => void }) {
  const parts = path.replace(/^\//, "").split("/").filter(Boolean);

  return (
    <nav className="flex items-center gap-1 text-sm flex-wrap">
      <button
        onClick={() => onNavigate("/")}
        className="flex items-center gap-1 text-muted-foreground hover:text-foreground"
      >
        <HardDrive className="h-3.5 w-3.5" />
        root
      </button>
      {parts.map((part, i) => {
        const to = "/" + parts.slice(0, i + 1).join("/");
        return (
          <span key={to} className="flex items-center gap-1">
            <ChevronRight className="h-3 w-3 text-muted-foreground" />
            <button
              onClick={() => onNavigate(to)}
              className={cn(
                "hover:text-foreground",
                i === parts.length - 1 ? "font-medium text-foreground" : "text-muted-foreground",
              )}
            >
              {part}
            </button>
          </span>
        );
      })}
    </nav>
  );
}

// ─── Context menu ─────────────────────────────────────────────────────────────

function ContextMenu({
  item,
  onRename,
  onDelete,
  onEdit,
}: {
  item: FileItem;
  onRename: (item: FileItem) => void;
  onDelete: (item: FileItem) => void;
  onEdit: (item: FileItem) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const isEditable = !item.is_dir && (TEXT_EXTS.has(item.ext) || CODE_EXTS.has(item.ext));

  return (
    <div ref={ref} className="relative">
      <button
        onClick={(e) => { e.stopPropagation(); setOpen((v) => !v); }}
        className="rounded p-1 opacity-0 group-hover:opacity-100 hover:bg-accent"
      >
        <MoreVertical className="h-4 w-4" />
      </button>
      {open && (
        <div className="absolute right-0 top-7 z-50 w-40 rounded-md border bg-background shadow-lg py-1">
          {isEditable && (
            <button
              className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
              onClick={() => { setOpen(false); onEdit(item); }}
            >
              <FileText className="h-3.5 w-3.5" /> Edit
            </button>
          )}
          {!item.is_dir && (
            <a
              href={fmDownloadUrl(item.path)}
              download={item.name}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
              onClick={() => setOpen(false)}
            >
              <Download className="h-3.5 w-3.5" /> Download
            </a>
          )}
          <button
            className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
            onClick={() => { setOpen(false); onRename(item); }}
          >
            <Pencil className="h-3.5 w-3.5" /> Rename
          </button>
          <button
            className="flex w-full items-center gap-2 px-3 py-2 text-sm text-destructive hover:bg-accent"
            onClick={() => { setOpen(false); onDelete(item); }}
          >
            <Trash2 className="h-3.5 w-3.5" /> Delete
          </button>
        </div>
      )}
    </div>
  );
}

// ─── Editor modal ─────────────────────────────────────────────────────────────

function EditorModal({
  item,
  onClose,
}: {
  item: FileItem;
  onClose: () => void;
}) {
  const [content, setContent] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    fmReadFile(item.path)
      .then((r) => setContent(r.content))
      .catch((e) => toast.error(e.message))
      .finally(() => setLoading(false));
  }, [item.path]);

  async function save() {
    setSaving(true);
    try {
      await fmWriteFile(item.path, content);
      toast.success("Saved");
      onClose();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="flex h-[80vh] w-full max-w-4xl flex-col rounded-lg border bg-background shadow-xl">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div className="flex items-center gap-2">
            <FileCode className="h-4 w-4 text-muted-foreground" />
            <span className="font-mono text-sm">{item.path}</span>
          </div>
          <div className="flex items-center gap-2">
            <Button size="sm" onClick={save} disabled={saving || loading}>
              {saving ? <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" /> : <Save className="mr-2 h-3.5 w-3.5" />}
              Save
            </Button>
            <Button size="sm" variant="ghost" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>
        {loading ? (
          <div className="flex flex-1 items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <textarea
            className="flex-1 resize-none bg-muted/30 p-4 font-mono text-sm outline-none"
            value={content}
            onChange={(e) => setContent(e.target.value)}
            spellCheck={false}
          />
        )}
      </div>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function FilesPage() {
  const qc = useQueryClient();
  const [currentPath, setCurrentPath] = useState("/");
  const [dragging, setDragging] = useState(false);

  // Modals / dialogs state
  const [renameItem, setRenameItem]     = useState<FileItem | null>(null);
  const [newName, setNewName]           = useState("");
  const [deleteItem, setDeleteItem]     = useState<FileItem | null>(null);
  const [editItem, setEditItem]         = useState<FileItem | null>(null);
  const [newDirOpen, setNewDirOpen]     = useState(false);
  const [newDirName, setNewDirName]     = useState("");
  const [uploading, setUploading]       = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const queryKey = ["files", currentPath];

  const { data, isLoading, isError } = useQuery({
    queryKey,
    queryFn: () => fmListDir(currentPath),
  });

  const deleteMutation = useMutation({
    mutationFn: (path: string) => fmDelete(path),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey });
      toast.success("Deleted");
      setDeleteItem(null);
    },
    onError: (e) => { toast.error(e instanceof Error ? e.message : "Delete failed"); setDeleteItem(null); },
  });

  const renameMutation = useMutation({
    mutationFn: ({ old_path, new_path }: { old_path: string; new_path: string }) =>
      fmRename(old_path, new_path),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey });
      toast.success("Renamed");
      setRenameItem(null);
      setNewName("");
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Rename failed"),
  });

  const mkdirMutation = useMutation({
    mutationFn: (name: string) => {
      const full = currentPath.replace(/\/$/, "") + "/" + name;
      return fmCreateDir(full);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey });
      toast.success("Folder created");
      setNewDirOpen(false);
      setNewDirName("");
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed"),
  });

  // ── Upload ────────────────────────────────────────────────────────────────

  const handleFiles = useCallback(async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    setUploading(true);
    try {
      await fmUpload(currentPath, Array.from(files));
      qc.invalidateQueries({ queryKey });
      toast.success(`Uploaded ${files.length} file(s)`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Upload failed");
    } finally {
      setUploading(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPath]);

  // Drag & drop handlers
  const onDragOver = (e: React.DragEvent) => { e.preventDefault(); setDragging(true); };
  const onDragLeave = () => setDragging(false);
  const onDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    handleFiles(e.dataTransfer.files);
  };

  const navigate = (path: string) => setCurrentPath(path);

  const goUp = () => {
    if (currentPath === "/") return;
    const parent = currentPath.replace(/\/[^/]+\/?$/, "") || "/";
    setCurrentPath(parent);
  };

  const items = data?.items ?? [];

  return (
    <div className="flex h-full flex-col space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <HardDrive className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">File Manager</h2>
            <p className="text-xs text-muted-foreground font-mono">
              root: {data?.root ?? "…"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setNewDirOpen(true)}
          >
            <FolderPlus className="mr-2 h-4 w-4" /> New Folder
          </Button>
          <Button
            size="sm"
            disabled={uploading}
            onClick={() => fileInputRef.current?.click()}
          >
            {uploading ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Upload className="mr-2 h-4 w-4" />
            )}
            Upload
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={(e) => handleFiles(e.target.files)}
          />
        </div>
      </div>

      {/* Breadcrumbs + back */}
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          disabled={currentPath === "/"}
          onClick={goUp}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <Breadcrumbs path={currentPath} onNavigate={navigate} />
      </div>

      {/* Drop zone + file list */}
      <Card
        className={cn(
          "flex-1 overflow-hidden transition-colors",
          dragging && "border-primary bg-primary/5",
        )}
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
      >
        {dragging && (
          <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center rounded-lg border-2 border-dashed border-primary bg-primary/10 text-primary text-sm font-medium">
            Drop files to upload
          </div>
        )}
        <CardContent className="p-0">
          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          )}
          {isError && (
            <p className="px-4 py-6 text-sm text-destructive">
              Failed to load directory.
            </p>
          )}
          {!isLoading && !isError && items.length === 0 && (
            <p className="px-4 py-8 text-center text-sm text-muted-foreground">
              Empty directory. Drop files here to upload.
            </p>
          )}
          {!isLoading && items.length > 0 && (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-xs text-muted-foreground">
                  <th className="px-4 py-2 text-left font-medium">Name</th>
                  <th className="px-4 py-2 text-right font-medium hidden sm:table-cell">Size</th>
                  <th className="px-4 py-2 text-right font-medium hidden md:table-cell">Modified</th>
                  <th className="w-10" />
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr
                    key={item.path}
                    className="group border-b last:border-0 hover:bg-muted/50 cursor-pointer"
                    onClick={() => {
                      if (item.is_dir) navigate(item.path);
                    }}
                  >
                    <td className="flex items-center gap-3 px-4 py-2.5">
                      <FileIcon item={item} className="h-4 w-4" />
                      <span className="truncate font-medium">{item.name}</span>
                    </td>
                    <td className="px-4 py-2.5 text-right text-muted-foreground hidden sm:table-cell">
                      {item.is_dir ? "—" : formatSize(item.size)}
                    </td>
                    <td className="px-4 py-2.5 text-right text-muted-foreground hidden md:table-cell">
                      {formatDistanceToNow(new Date(item.mod_time), { addSuffix: true })}
                    </td>
                    <td className="px-2 py-2.5" onClick={(e) => e.stopPropagation()}>
                      <ContextMenu
                        item={item}
                        onRename={(i) => { setRenameItem(i); setNewName(i.name); }}
                        onDelete={setDeleteItem}
                        onEdit={setEditItem}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {/* ── Rename dialog ── */}
      {renameItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-sm rounded-lg border bg-background p-6 shadow-xl space-y-4">
            <h3 className="font-semibold">Rename</h3>
            <div className="space-y-1">
              <Label>New name</Label>
              <Input
                autoFocus
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && newName.trim()) {
                    const dir = renameItem.path.substring(0, renameItem.path.lastIndexOf("/")) || "/";
                    renameMutation.mutate({
                      old_path: renameItem.path,
                      new_path: dir + "/" + newName.trim(),
                    });
                  }
                }}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => { setRenameItem(null); setNewName(""); }}>
                Cancel
              </Button>
              <Button
                disabled={!newName.trim() || renameMutation.isPending}
                onClick={() => {
                  const dir = renameItem.path.substring(0, renameItem.path.lastIndexOf("/")) || "/";
                  renameMutation.mutate({
                    old_path: renameItem.path,
                    new_path: dir + "/" + newName.trim(),
                  });
                }}
              >
                {renameMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Rename
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* ── New folder dialog ── */}
      {newDirOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-sm rounded-lg border bg-background p-6 shadow-xl space-y-4">
            <h3 className="font-semibold">New Folder</h3>
            <div className="space-y-1">
              <Label>Folder name</Label>
              <Input
                autoFocus
                placeholder="my-folder"
                value={newDirName}
                onChange={(e) => setNewDirName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && newDirName.trim() && mkdirMutation.mutate(newDirName.trim())}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => { setNewDirOpen(false); setNewDirName(""); }}>
                Cancel
              </Button>
              <Button
                disabled={!newDirName.trim() || mkdirMutation.isPending}
                onClick={() => mkdirMutation.mutate(newDirName.trim())}
              >
                {mkdirMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Create
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* ── Delete confirm ── */}
      <ConfirmDialog
        open={!!deleteItem}
        title={`Delete "${deleteItem?.name}"?`}
        description={
          deleteItem?.is_dir
            ? "This will delete the folder and all its contents. This cannot be undone."
            : "This file will be permanently deleted."
        }
        loading={deleteMutation.isPending}
        onConfirm={() => deleteItem && deleteMutation.mutate(deleteItem.path)}
        onCancel={() => setDeleteItem(null)}
      />

      {/* ── File editor ── */}
      {editItem && (
        <EditorModal item={editItem} onClose={() => { setEditItem(null); qc.invalidateQueries({ queryKey }); }} />
      )}
    </div>
  );
}
