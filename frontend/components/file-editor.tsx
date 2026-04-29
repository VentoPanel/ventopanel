"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import Editor, { type OnMount } from "@monaco-editor/react";
import type * as MonacoType from "monaco-editor";
import { Save, X, Loader2, FileCode } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { fmReadFile, fmWriteFile, type FileItem } from "@/lib/api";

// ─── Language detector ────────────────────────────────────────────────────────

const EXT_TO_LANG: Record<string, string> = {
  ".go":    "go",
  ".js":    "javascript",
  ".mjs":   "javascript",
  ".cjs":   "javascript",
  ".ts":    "typescript",
  ".tsx":   "typescript",
  ".jsx":   "javascript",
  ".py":    "python",
  ".html":  "html",
  ".htm":   "html",
  ".css":   "css",
  ".scss":  "scss",
  ".less":  "less",
  ".json":  "json",
  ".yaml":  "yaml",
  ".yml":   "yaml",
  ".toml":  "ini",
  ".ini":   "ini",
  ".sh":    "shell",
  ".bash":  "shell",
  ".zsh":   "shell",
  ".env":   "shell",
  ".sql":   "sql",
  ".xml":   "xml",
  ".md":    "markdown",
  ".mdx":   "markdown",
  ".rs":    "rust",
  ".php":   "php",
  ".rb":    "ruby",
  ".java":  "java",
  ".c":     "c",
  ".cpp":   "cpp",
  ".h":     "cpp",
  ".cs":    "csharp",
  ".kt":    "kotlin",
  ".swift": "swift",
  ".tf":    "hcl",
  ".hcl":   "hcl",
  ".lua":   "lua",
  ".r":     "r",
  ".dockerfile": "dockerfile",
};

function detectLanguage(path: string): string {
  const lower = path.toLowerCase();
  // Special full-name matches.
  const base = lower.split("/").pop() ?? "";
  if (base === "dockerfile")         return "dockerfile";
  if (base === "makefile")           return "makefile";
  if (base === ".gitignore")         return "shell";
  if (base === ".env" || base.startsWith(".env.")) return "shell";

  const dot = base.lastIndexOf(".");
  if (dot === -1) return "plaintext";
  return EXT_TO_LANG[base.slice(dot)] ?? "plaintext";
}

// ─── Props ────────────────────────────────────────────────────────────────────

interface Props {
  item: FileItem;
  onClose: () => void;
  onSaved?: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────────

export function FileEditor({ item, onClose, onSaved }: Props) {
  const [content,  setContent]  = useState<string | null>(null);
  const [loading,  setLoading]  = useState(true);
  const [saving,   setSaving]   = useState(false);
  const [dirty,    setDirty]    = useState(false);
  const editorRef  = useRef<MonacoType.editor.IStandaloneCodeEditor | null>(null);
  const monacoRef  = useRef<typeof MonacoType | null>(null);

  const lang = detectLanguage(item.path);

  // Load file content.
  useEffect(() => {
    setLoading(true);
    fmReadFile(item.path)
      .then((r) => { setContent(r.content); setDirty(false); })
      .catch((e) => toast.error(e instanceof Error ? e.message : "Failed to load file"))
      .finally(() => setLoading(false));
  }, [item.path]);

  // Save handler.
  const save = useCallback(async () => {
    const value = editorRef.current?.getValue() ?? content ?? "";
    setSaving(true);
    try {
      await fmWriteFile(item.path, value);
      setDirty(false);
      toast.success("File saved");
      onSaved?.();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }, [content, item.path, onSaved]);

  // Register Ctrl+S / Cmd+S via Monaco action.
  const handleMount: OnMount = (editor, monaco) => {
    editorRef.current  = editor;
    monacoRef.current  = monaco;

    editor.addAction({
      id:    "save-file",
      label: "Save File",
      keybindings: [
        monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS,
      ],
      run: () => { save(); },
    });

    // Track dirty state.
    editor.onDidChangeModelContent(() => setDirty(true));
  };

  // Close guard: warn if dirty.
  function handleClose() {
    if (dirty) {
      if (!window.confirm("You have unsaved changes. Close anyway?")) return;
    }
    onClose();
  }

  // Global keydown for Escape.
  useEffect(() => {
    function handler(e: KeyboardEvent) {
      if (e.key === "Escape") handleClose();
    }
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dirty]);

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-[#1e1e1e]">

      {/* ── Toolbar ── */}
      <div className="flex items-center justify-between border-b border-white/10 bg-[#252526] px-4 py-2">
        <div className="flex items-center gap-3 min-w-0">
          <FileCode className="h-4 w-4 text-blue-400 shrink-0" />
          <span className="font-mono text-sm text-white/80 truncate" title={item.path}>
            {item.path}
          </span>
          {dirty && (
            <span className="text-xs text-amber-400 shrink-0">● unsaved</span>
          )}
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {/* Language badge */}
          <span className="hidden sm:inline-flex rounded bg-white/10 px-2 py-0.5 text-xs text-white/60 font-mono">
            {lang}
          </span>

          <Button
            size="sm"
            onClick={save}
            disabled={saving || loading}
            className="bg-blue-600 hover:bg-blue-700 text-white border-0"
          >
            {saving
              ? <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
              : <Save className="mr-2 h-3.5 w-3.5" />
            }
            {saving ? "Saving…" : "Save"}
          </Button>

          <Button
            size="sm"
            variant="ghost"
            onClick={handleClose}
            className="text-white/60 hover:text-white hover:bg-white/10 border-0"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* ── Editor ── */}
      {loading ? (
        <div className="flex flex-1 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-white/40" />
        </div>
      ) : (
        <div className="flex-1">
          <Editor
            height="100%"
            theme="vs-dark"
            language={lang}
            value={content ?? ""}
            onMount={handleMount}
            onChange={(v) => { setContent(v ?? ""); setDirty(true); }}
            options={{
              fontSize: 14,
              fontFamily: "'Fira Code', 'Cascadia Code', Menlo, Monaco, monospace",
              fontLigatures: true,
              minimap:    { enabled: true },
              wordWrap:   "on",
              lineNumbers: "on",
              scrollBeyondLastLine: false,
              automaticLayout: true,
              tabSize: 2,
              renderWhitespace: "boundary",
              bracketPairColorization: { enabled: true },
              smoothScrolling: true,
              cursorBlinking: "smooth",
              padding: { top: 12 },
            }}
          />
        </div>
      )}
    </div>
  );
}
