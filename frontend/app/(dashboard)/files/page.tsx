"use client";

import {
  useState,
  useRef,
  useCallback,
  useEffect,
  Suspense,
} from "react";
import { useRouter, useSearchParams } from "next/navigation";
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
  X,
  Loader2,
  HardDrive,
  ChevronRight,
  Archive,
  PackageOpen,
  Lock,
  CheckSquare,
  Server,
} from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fmListDir,
  fmDelete,
  fmCreateDir,
  fmRename,
  fmUpload,
  fmDownloadUrl,
  fmCompress,
  fmExtract,
  fetchServers,
  type FileItem,
  type Server,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { FileEditor } from "@/components/file-editor";
import { PermissionsModal } from "@/components/permissions-modal";
import { toast } from "sonner";
import { formatDistanceToNow } from "date-fns";
import { cn } from "@/lib/utils";

// ─── Constants ────────────────────────────────────────────────────────────────

const TEXT_EXTS = new Set([".txt", ".md", ".log", ".csv", ".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".env"]);
const CODE_EXTS = new Set([".js", ".ts", ".tsx", ".jsx", ".go", ".py", ".php", ".rb", ".rs", ".sh", ".bash", ".css", ".html", ".htm", ".sql"]);
const IMG_EXTS  = new Set([".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico"]);

function isEditable(item: FileItem) {
  return !item.is_dir && (TEXT_EXTS.has(item.ext) || CODE_EXTS.has(item.ext));
}

// ─── File icon ────────────────────────────────────────────────────────────────

function FileIcon({ item, className }: { item: FileItem; className?: string }) {
  const cls = cn("shrink-0", className);
  if (item.is_dir)               return <Folder    className={cn(cls, "text-amber-400")} />;
  if (CODE_EXTS.has(item.ext))   return <FileCode  className={cn(cls, "text-blue-400")} />;
  if (TEXT_EXTS.has(item.ext))   return <FileText  className={cn(cls, "text-gray-400")} />;
  if (IMG_EXTS.has(item.ext))    return <FileImage className={cn(cls, "text-pink-400")} />;
  return <File className={cn(cls, "text-gray-400")} />;
}

function formatSize(bytes: number): string {
  if (bytes < 1024)        return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

// ─── Breadcrumbs ─────────────────────────────────────────────────────────────

function Breadcrumbs({ path, onNavigate }: { path: string; onNavigate: (p: string) => void }) {
  const parts = path.replace(/^\//, "").split("/").filter(Boolean);
  return (
    <nav className="flex items-center gap-1 text-sm flex-wrap min-w-0">
      <button
        onClick={() => onNavigate("/")}
        className="flex items-center gap-1 text-muted-foreground hover:text-foreground shrink-0"
      >
        <HardDrive className="h-3.5 w-3.5" />
        root
      </button>
      {parts.map((part, i) => {
        const to = "/" + parts.slice(0, i + 1).join("/");
        return (
          <span key={to} className="flex items-center gap-1 min-w-0">
            <ChevronRight className="h-3 w-3 text-muted-foreground shrink-0" />
            <button
              onClick={() => onNavigate(to)}
              className={cn(
                "hover:text-foreground truncate max-w-[160px]",
                i === parts.length - 1
                  ? "font-medium text-foreground"
                  : "text-muted-foreground",
              )}
              title={part}
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
  serverId,
  onRename,
  onDelete,
  onEdit,
  onCompress,
  onExtract,
  onPermissions,
}: {
  item: FileItem;
  serverId?: string;
  onRename: (item: FileItem) => void;
  onDelete: (item: FileItem) => void;
  onEdit: (item: FileItem) => void;
  onCompress: (item: FileItem) => void;
  onExtract: (item: FileItem) => void;
  onPermissions: (item: FileItem) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function h(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", h);
    return () => document.removeEventListener("mousedown", h);
  }, []);

  const editable = isEditable(item);
  const isZip    = item.ext === ".zip";

  return (
    <div ref={ref} className="relative">
      <button
        onClick={(e) => { e.stopPropagation(); setOpen((v) => !v); }}
        className="rounded p-1 opacity-0 group-hover:opacity-100 hover:bg-accent transition-opacity"
      >
        <MoreVertical className="h-4 w-4" />
      </button>
      {open && (
        <div className="absolute right-0 top-7 z-50 w-44 rounded-md border bg-background shadow-lg py-1">
          {editable && (
            <button className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
              onClick={() => { setOpen(false); onEdit(item); }}>
              <FileText className="h-3.5 w-3.5" /> Edit
            </button>
          )}
          <a
            href={fmDownloadUrl(item.path, serverId)}
            download={item.is_dir ? item.name + ".zip" : item.name}
            className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
            onClick={() => setOpen(false)}
          >
            <Download className="h-3.5 w-3.5" />
            {item.is_dir ? "Download as ZIP" : "Download"}
          </a>
          <button className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
            onClick={() => { setOpen(false); onCompress(item); }}>
            <Archive className="h-3.5 w-3.5" /> Compress to ZIP
          </button>
          {isZip && (
            <button className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
              onClick={() => { setOpen(false); onExtract(item); }}>
              <PackageOpen className="h-3.5 w-3.5" /> Extract here
            </button>
          )}
          <button className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
            onClick={() => { setOpen(false); onPermissions(item); }}>
            <Lock className="h-3.5 w-3.5" /> Permissions
          </button>
          <div className="my-1 border-t" />
          <button className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
            onClick={() => { setOpen(false); onRename(item); }}>
            <Pencil className="h-3.5 w-3.5" /> Rename
          </button>
          <button className="flex w-full items-center gap-2 px-3 py-2 text-sm text-destructive hover:bg-accent"
            onClick={() => { setOpen(false); onDelete(item); }}>
            <Trash2 className="h-3.5 w-3.5" /> Delete
          </button>
        </div>
      )}
    </div>
  );
}


// ─── Floating Action Bar ──────────────────────────────────────────────────────

function FloatingBar({
  selected,
  currentPath,
  serverId,
  onClearSelection,
  onCompressSelected,
  onDeleteSelected,
}: {
  selected: FileItem[];
  currentPath: string;
  serverId?: string;
  onClearSelection: () => void;
  onCompressSelected: (items: FileItem[]) => void;
  onDeleteSelected: (items: FileItem[]) => void;
}) {
  if (selected.length === 0) return null;

  const downloadHref = selected.length === 1
    ? fmDownloadUrl(selected[0].path, serverId)
    : fmDownloadUrl(currentPath, serverId);

  return (
    <div className="fixed bottom-6 left-1/2 z-40 -translate-x-1/2 animate-in slide-in-from-bottom-4 fade-in duration-200">
      <div className="flex items-center gap-2 rounded-full border bg-background/95 backdrop-blur shadow-xl px-4 py-2">
        <span className="text-sm font-medium pl-1 pr-2 text-muted-foreground">
          {selected.length} selected
        </span>
        <div className="h-4 w-px bg-border" />
        <a
          href={downloadHref}
          download
          className="flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium hover:bg-accent transition-colors"
        >
          <Download className="h-3.5 w-3.5" />
          {selected.length === 1 && !selected[0].is_dir ? "Download" : "Download ZIP"}
        </a>
        <button
          className="flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium hover:bg-accent transition-colors"
          onClick={() => onCompressSelected(selected)}
        >
          <Archive className="h-3.5 w-3.5" /> Compress
        </button>
        <button
          className="flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors"
          onClick={() => onDeleteSelected(selected)}
        >
          <Trash2 className="h-3.5 w-3.5" /> Delete
        </button>
        <div className="h-4 w-px bg-border" />
        <button
          className="rounded-full p-1.5 hover:bg-accent transition-colors text-muted-foreground"
          onClick={onClearSelection}
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

function FilesPageInner() {
  const router       = useRouter();
  const searchParams = useSearchParams();
  const qc           = useQueryClient();

  // Both path and server_id live in the URL so Back/Forward works.
  const currentPath = searchParams.get("path") ?? "/";
  const serverId    = searchParams.get("server_id") ?? "";

  const navigate = useCallback(
    (p: string, sid?: string) => {
      const s = sid !== undefined ? sid : serverId;
      const url = `/files?path=${encodeURIComponent(p)}` + (s ? `&server_id=${encodeURIComponent(s)}` : "");
      router.push(url);
    },
    [router, serverId],
  );

  // Change server → reset path to "/"
  const handleServerChange = useCallback(
    (sid: string) => navigate("/", sid === "__local__" ? "" : sid),
    [navigate],
  );

  // Servers list for the selector.
  const { data: serversData } = useQuery({
    queryKey: ["servers"],
    queryFn: fetchServers,
    staleTime: 30_000,
  });

  const [dragging, setDragging] = useState(false);

  // ── Selection ────────────────────────────────────────────────────────────
  const [selected, setSelected]   = useState<Set<string>>(new Set());
  const lastClickedIdx            = useRef<number | null>(null);

  // Reset selection when changing directory.
  useEffect(() => {
    setSelected(new Set());
    lastClickedIdx.current = null;
  }, [currentPath]);

  // ── Modals ───────────────────────────────────────────────────────────────
  const [renameItem,    setRenameItem]    = useState<FileItem | null>(null);
  const [newName,       setNewName]       = useState("");
  const [deleteItems,   setDeleteItems]   = useState<FileItem[]>([]);
  const [editItem,      setEditItem]      = useState<FileItem | null>(null);
  const [newDirOpen,    setNewDirOpen]    = useState(false);
  const [newDirName,    setNewDirName]    = useState("");
  const [uploading,     setUploading]     = useState(false);
  const [compressItems, setCompressItems] = useState<FileItem[]>([]);
  const [compressName,  setCompressName]  = useState("");
  const [permItem,      setPermItem]      = useState<FileItem | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const queryKey = ["files", currentPath, serverId];

  const { data, isLoading, isError } = useQuery({
    queryKey,
    queryFn: () => fmListDir(currentPath, serverId || undefined),
  });

  const items = data?.items ?? [];

  // ── Mutations ────────────────────────────────────────────────────────────

  const sid = serverId || undefined;

  const deleteMutation = useMutation({
    mutationFn: (paths: string[]) => Promise.all(paths.map((p) => fmDelete(p, sid))),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey });
      toast.success("Deleted");
      setDeleteItems([]);
      setSelected(new Set());
    },
    onError: (e) => { toast.error(e instanceof Error ? e.message : "Delete failed"); setDeleteItems([]); },
  });

  const renameMutation = useMutation({
    mutationFn: ({ old_path, new_path }: { old_path: string; new_path: string }) =>
      fmRename(old_path, new_path, sid),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey });
      toast.success("Renamed");
      setRenameItem(null); setNewName("");
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Rename failed"),
  });

  const mkdirMutation = useMutation({
    mutationFn: (name: string) => fmCreateDir(currentPath.replace(/\/$/, "") + "/" + name, sid),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey });
      toast.success("Folder created");
      setNewDirOpen(false); setNewDirName("");
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed"),
  });

  const compressMutation = useMutation({
    mutationFn: ({ srcPaths, name }: { srcPaths: string[]; name: string }) => {
      const dest = currentPath.replace(/\/$/, "") + "/" + name + ".zip";
      return fmCompress(srcPaths, dest, sid);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey });
      toast.success("Compressed");
      setCompressItems([]); setCompressName(""); setSelected(new Set());
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed"),
  });

  const extractMutation = useMutation({
    mutationFn: (path: string) => fmExtract(path, currentPath, sid),
    onSuccess: () => { qc.invalidateQueries({ queryKey }); toast.success("Extracted"); },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Extract failed"),
  });

  // ── Upload ────────────────────────────────────────────────────────────────

  const handleFiles = useCallback(async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    setUploading(true);
    try {
      await fmUpload(currentPath, Array.from(files), sid);
      qc.invalidateQueries({ queryKey });
      toast.success(`Uploaded ${files.length} file(s)`);
    } catch (e) { toast.error(e instanceof Error ? e.message : "Upload failed"); }
    finally { setUploading(false); }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPath]);

  // ── Selection helpers ─────────────────────────────────────────────────────

  function toggleItem(item: FileItem, idx: number, shiftHeld: boolean) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (shiftHeld && lastClickedIdx.current !== null) {
        // Range selection: toggle all items between last click and current.
        const lo = Math.min(lastClickedIdx.current, idx);
        const hi = Math.max(lastClickedIdx.current, idx);
        const shouldSelect = !prev.has(item.path);
        for (let i = lo; i <= hi; i++) {
          if (shouldSelect) next.add(items[i].path);
          else next.delete(items[i].path);
        }
      } else {
        if (next.has(item.path)) next.delete(item.path);
        else next.add(item.path);
      }
      return next;
    });
    lastClickedIdx.current = idx;
  }

  function toggleAll() {
    if (selected.size === items.length) setSelected(new Set());
    else setSelected(new Set(items.map((i) => i.path)));
  }

  const allChecked = items.length > 0 && selected.size === items.length;
  const someChecked = selected.size > 0 && !allChecked;
  const selectedItems = items.filter((i) => selected.has(i.path));

  const goUp = () => {
    if (currentPath === "/") return;
    const parent = currentPath.replace(/\/[^/]+\/?$/, "") || "/";
    navigate(parent);
  };

  // ── Drag & Drop ──────────────────────────────────────────────────────────
  const onDragOver  = (e: React.DragEvent) => { e.preventDefault(); setDragging(true); };
  const onDragLeave = (e: React.DragEvent) => {
    // Only clear when leaving the card entirely (not a child element).
    if (!e.currentTarget.contains(e.relatedTarget as Node)) setDragging(false);
  };
  const onDrop = (e: React.DragEvent) => {
    e.preventDefault(); setDragging(false);
    handleFiles(e.dataTransfer.files);
  };

  return (
    <div className="flex h-full flex-col gap-4">

      {/* ── Header ── */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <HardDrive className="h-6 w-6 text-muted-foreground shrink-0" />
          <div className="min-w-0">
            <h2 className="text-2xl font-bold tracking-tight">File Manager</h2>
            <p className="text-xs text-muted-foreground font-mono truncate">{data?.root ?? "…"}</p>
          </div>
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          {/* ── Server selector ── */}
          <div className="relative flex items-center">
            <Server className="pointer-events-none absolute left-2.5 h-3.5 w-3.5 text-muted-foreground" />
            <select
              value={serverId || "__local__"}
              onChange={(e) => handleServerChange(e.target.value)}
              className="h-8 rounded-md border border-input bg-background pl-8 pr-8 text-xs font-medium shadow-sm focus:outline-none focus:ring-1 focus:ring-ring appearance-none cursor-pointer hover:bg-accent transition-colors min-w-[170px]"
            >
              <option value="__local__">⬡ Local (panel host)</option>
              {serversData?.map((srv: Server) => (
                <option key={srv.ID} value={srv.ID}>
                  {srv.Name} ({srv.Host})
                </option>
              ))}
            </select>
            <ChevronRight className="pointer-events-none absolute right-2.5 h-3.5 w-3.5 rotate-90 text-muted-foreground" />
          </div>

          <Button variant="outline" size="sm" onClick={() => setNewDirOpen(true)}>
            <FolderPlus className="mr-2 h-4 w-4" /> New Folder
          </Button>
          <Button size="sm" disabled={uploading} onClick={() => fileInputRef.current?.click()}>
            {uploading
              ? <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              : <Upload className="mr-2 h-4 w-4" />}
            Upload
          </Button>
          <input ref={fileInputRef} type="file" multiple className="hidden"
            onChange={(e) => handleFiles(e.target.files)} />
        </div>
      </div>

      {/* ── Breadcrumbs + Back ── */}
      <div className="flex items-center gap-3 min-w-0">
        <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0"
          disabled={currentPath === "/"} onClick={goUp}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <Breadcrumbs path={currentPath} onNavigate={navigate} />
      </div>

      {/* ── File table with D&D zone ── */}
      <Card
        className={cn(
          "flex-1 overflow-auto transition-all duration-150 relative",
          dragging && "border-2 border-dashed border-primary bg-primary/5",
        )}
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
      >
        {/* D&D overlay */}
        {dragging && (
          <div className="pointer-events-none absolute inset-0 z-20 flex flex-col items-center justify-center gap-2 rounded-lg text-primary">
            <Upload className="h-10 w-10 opacity-60" />
            <span className="text-sm font-medium">Drop files to upload</span>
          </div>
        )}

        <CardContent className="p-0">
          {isLoading && (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          )}
          {isError && (
            <p className="px-4 py-6 text-sm text-destructive">Failed to load directory.</p>
          )}
          {!isLoading && !isError && items.length === 0 && (
            <div className="flex flex-col items-center justify-center py-20 gap-4 text-muted-foreground">
              <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/50">
                <Folder className="h-8 w-8 opacity-40" />
              </div>
              <div className="text-center">
                <p className="font-medium text-foreground">Empty folder</p>
                <p className="text-sm mt-1">Drop files here or upload your first file</p>
              </div>
              <Button
                size="sm"
                variant="outline"
                onClick={() => fileInputRef.current?.click()}
              >
                <Upload className="mr-2 h-3.5 w-3.5" />
                Upload first file
              </Button>
            </div>
          )}

          {!isLoading && items.length > 0 && (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-xs text-muted-foreground">
                  <th className="w-10 px-3 py-2">
                    {/* Select All checkbox */}
                    <input
                      type="checkbox"
                      checked={allChecked}
                      ref={(el) => { if (el) el.indeterminate = someChecked; }}
                      onChange={toggleAll}
                      className="cursor-pointer accent-primary"
                      aria-label="Select all"
                    />
                  </th>
                  <th className="px-3 py-2 text-left font-medium">Name</th>
                  <th className="px-3 py-2 text-right font-medium hidden sm:table-cell">Size</th>
                  <th className="px-3 py-2 text-right font-medium hidden md:table-cell">Modified</th>
                  <th className="w-10" />
                </tr>
              </thead>
              <tbody>
                {items.map((item, idx) => {
                  const isSelected = selected.has(item.path);
                  return (
                    <tr
                      key={item.path}
                      className={cn(
                        "group border-b last:border-0 transition-colors",
                        isSelected
                          ? "bg-primary/8 hover:bg-primary/12"
                          : "hover:bg-muted/50",
                      )}
                    >
                      {/* Checkbox cell — does NOT navigate */}
                      <td
                        className="px-3 py-2.5"
                        onClick={(e) => { e.stopPropagation(); toggleItem(item, idx, e.shiftKey); }}
                      >
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => {}}
                          className="cursor-pointer accent-primary"
                          aria-label={`Select ${item.name}`}
                        />
                      </td>

                      {/* Name cell — navigates into dir on click */}
                      <td
                        className="flex items-center gap-3 px-3 py-2.5 cursor-pointer"
                        onClick={() => item.is_dir && navigate(item.path)}
                      >
                        <FileIcon item={item} className="h-4 w-4" />
                        <span className={cn("truncate font-medium", item.is_dir && "hover:underline")}>
                          {item.name}
                        </span>
                      </td>

                      <td className="px-3 py-2.5 text-right text-muted-foreground hidden sm:table-cell">
                        {item.is_dir ? "—" : formatSize(item.size)}
                      </td>
                      <td className="px-3 py-2.5 text-right text-muted-foreground hidden md:table-cell">
                        {formatDistanceToNow(new Date(item.mod_time), { addSuffix: true })}
                      </td>
                      <td className="px-2 py-2.5" onClick={(e) => e.stopPropagation()}>
                        <ContextMenu
                          item={item}
                          serverId={sid}
                          onRename={(i) => { setRenameItem(i); setNewName(i.name); }}
                          onDelete={(i) => setDeleteItems([i])}
                          onEdit={setEditItem}
                          onCompress={(i) => { setCompressItems([i]); setCompressName(i.name); }}
                          onExtract={(i) => extractMutation.mutate(i.path)}
                          onPermissions={(i) => setPermItem(i)}
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {/* ── Floating Action Bar ── */}
      <FloatingBar
        selected={selectedItems}
        currentPath={currentPath}
        serverId={sid}
        onClearSelection={() => setSelected(new Set())}
        onCompressSelected={(its) => {
          setCompressItems(its);
          setCompressName(its.length === 1 ? its[0].name : "archive");
        }}
        onDeleteSelected={setDeleteItems}
      />

      {/* ── Rename dialog ── */}
      {renameItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-sm rounded-lg border bg-background p-6 shadow-xl space-y-4">
            <h3 className="font-semibold">Rename</h3>
            <div className="space-y-1">
              <Label>New name</Label>
              <Input
                autoFocus value={newName}
                onChange={(e) => setNewName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && newName.trim()) {
                    const dir = renameItem.path.substring(0, renameItem.path.lastIndexOf("/")) || "/";
                    renameMutation.mutate({ old_path: renameItem.path, new_path: dir + "/" + newName.trim() });
                  }
                }}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => { setRenameItem(null); setNewName(""); }}>Cancel</Button>
              <Button
                disabled={!newName.trim() || renameMutation.isPending}
                onClick={() => {
                  const dir = renameItem.path.substring(0, renameItem.path.lastIndexOf("/")) || "/";
                  renameMutation.mutate({ old_path: renameItem.path, new_path: dir + "/" + newName.trim() });
                }}
              >
                {renameMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />} Rename
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
                autoFocus placeholder="my-folder" value={newDirName}
                onChange={(e) => setNewDirName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && newDirName.trim() && mkdirMutation.mutate(newDirName.trim())}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => { setNewDirOpen(false); setNewDirName(""); }}>Cancel</Button>
              <Button disabled={!newDirName.trim() || mkdirMutation.isPending}
                onClick={() => mkdirMutation.mutate(newDirName.trim())}>
                {mkdirMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />} Create
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* ── Compress dialog ── */}
      {compressItems.length > 0 && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-sm rounded-lg border bg-background p-6 shadow-xl space-y-4">
            <h3 className="font-semibold flex items-center gap-2">
              <Archive className="h-4 w-4" /> Compress to ZIP
            </h3>
            {compressItems.length > 1 && (
              <p className="text-xs text-muted-foreground">
                {compressItems.length} items selected
              </p>
            )}
            <div className="space-y-1">
              <Label>Archive name (without .zip)</Label>
              <Input
                autoFocus value={compressName}
                onChange={(e) => setCompressName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && compressName.trim() &&
                  compressMutation.mutate({ srcPaths: compressItems.map(i => i.path), name: compressName.trim() })}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => { setCompressItems([]); setCompressName(""); }}>Cancel</Button>
              <Button
                disabled={!compressName.trim() || compressMutation.isPending}
                onClick={() => compressMutation.mutate({ srcPaths: compressItems.map(i => i.path), name: compressName.trim() })}
              >
                {compressMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />} Compress
              </Button>
            </div>
          </div>
        </div>
      )}


      {/* ── Delete confirm ── */}
      <ConfirmDialog
        open={deleteItems.length > 0}
        title={
          deleteItems.length === 1
            ? `Delete "${deleteItems[0]?.name}"?`
            : `Delete ${deleteItems.length} items?`
        }
        description={
          deleteItems.some((i) => i.is_dir)
            ? "Folders and all their contents will be permanently deleted."
            : "These files will be permanently deleted."
        }
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate(deleteItems.map((i) => i.path))}
        onCancel={() => setDeleteItems([])}
      />

      {/* ── File editor (Monaco) ── */}
      {editItem && (
        <FileEditor
          item={editItem}
          serverId={sid}
          onClose={() => setEditItem(null)}
          onSaved={() => qc.invalidateQueries({ queryKey })}
        />
      )}

      {/* ── Permissions modal (visual chmod) ── */}
      {permItem && (
        <PermissionsModal
          item={permItem}
          serverId={sid}
          onClose={() => setPermItem(null)}
        />
      )}

      {/* Unused import guard */}
      <span className="hidden"><CheckSquare className="h-0 w-0" /></span>
    </div>
  );
}

export default function FilesPage() {
  return (
    <Suspense fallback={
      <div className="flex h-full items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    }>
      <FilesPageInner />
    </Suspense>
  );
}
