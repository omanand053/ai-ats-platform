"use client";

import { useCallback, useRef, useState } from "react";
import type { CopilotUpload } from "@/services/copilot.service";
import {
  deleteCopilotUpload,
  importCopilotUpload,
  pasteCopilotText,
  uploadCopilotFile,
} from "@/services/copilot.service";

type Props = {
  sessionId: string;
  uploads: CopilotUpload[];
  onChange: () => void;
  onAnalyze: (uploadId: string, prompt: string) => void;
};

const ACCEPT = ".pdf,.docx,.txt,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain";

export default function CopilotUploadPanel({ sessionId, uploads, onChange, onAnalyze }: Props) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [busy, setBusy] = useState(false);
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteText, setPasteText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [importMode, setImportMode] = useState(false);

  const ingestFile = useCallback(
    async (file: File) => {
      setBusy(true);
      setError(null);
      try {
        await uploadCopilotFile(file, sessionId, importMode ? "import" : "temporary");
        onChange();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Upload failed");
      } finally {
        setBusy(false);
      }
    },
    [sessionId, importMode, onChange],
  );

  const onDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file) await ingestFile(file);
  };

  const submitPaste = async () => {
    if (!pasteText.trim()) return;
    setBusy(true);
    setError(null);
    try {
      await pasteCopilotText(pasteText, sessionId);
      setPasteText("");
      setPasteOpen(false);
      onChange();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Paste failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-3 border-b border-white/10 px-4 py-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-xs font-semibold text-white/90">Upload for analysis</p>
        <label className="flex items-center gap-1.5 text-[10px] text-white/70">
          <input
            type="checkbox"
            checked={importMode}
            onChange={(e) => setImportMode(e.target.checked)}
            className="rounded border-white/30"
          />
          Import to ATS
        </label>
      </div>

      <div
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={onDrop}
        className={`rounded-2xl border border-dashed px-3 py-4 text-center transition ${
          dragOver ? "border-white bg-white/15" : "border-white/25 bg-white/5"
        }`}
      >
        <p className="text-xs text-white/80">Drag & drop PDF, DOCX, or TXT</p>
        <div className="mt-2 flex flex-wrap items-center justify-center gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={() => inputRef.current?.click()}
            className="rounded-xl bg-white/15 px-3 py-1.5 text-[11px] font-semibold text-white hover:bg-white/25 disabled:opacity-50"
          >
            Browse files
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => setPasteOpen((v) => !v)}
            className="rounded-xl bg-white/10 px-3 py-1.5 text-[11px] font-medium text-white/90 hover:bg-white/20"
          >
            Paste text
          </button>
        </div>
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPT}
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) void ingestFile(file);
            e.target.value = "";
          }}
        />
        {busy ? <p className="mt-2 text-[11px] text-white/70">Processing… parsing · chunking · embedding</p> : null}
        {error ? <p className="mt-2 text-[11px] text-rose-200">{error}</p> : null}
      </div>

      {pasteOpen ? (
        <div className="space-y-2 rounded-2xl bg-black/20 p-2">
          <textarea
            value={pasteText}
            onChange={(e) => setPasteText(e.target.value)}
            rows={4}
            placeholder="Paste resume or JD text…"
            className="w-full rounded-xl border border-white/15 bg-black/20 px-3 py-2 text-xs text-white outline-none placeholder:text-white/40"
          />
          <div className="flex justify-end gap-2">
            <button type="button" onClick={() => setPasteOpen(false)} className="text-[11px] text-white/60">
              Cancel
            </button>
            <button
              type="button"
              disabled={busy || !pasteText.trim()}
              onClick={() => void submitPaste()}
              className="rounded-lg bg-white px-3 py-1 text-[11px] font-semibold text-slate-900 disabled:opacity-50"
            >
              Analyze
            </button>
          </div>
        </div>
      ) : null}

      {uploads.length > 0 ? (
        <div className="space-y-2">
          {uploads.map((u) => (
            <div key={u.id} className="rounded-2xl border border-white/10 bg-black/20 px-3 py-2">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate text-xs font-semibold text-white">{u.file_name}</p>
                  <p className="text-[10px] text-white/60">
                    {u.status}
                    {u.chunk_count != null ? ` · ${u.chunk_count} chunks` : ""}
                    {u.mode === "import" || u.status === "imported" ? " · ATS import" : " · temporary"}
                  </p>
                  {u.parsed_name ? (
                    <p className="mt-0.5 text-[10px] text-white/70">
                      {u.parsed_name}
                      {u.parsed_email ? ` · ${u.parsed_email}` : ""}
                    </p>
                  ) : null}
                </div>
                <button
                  type="button"
                  className="text-[10px] text-white/50 hover:text-white"
                  onClick={async () => {
                    await deleteCopilotUpload(u.id);
                    onChange();
                  }}
                >
                  Remove
                </button>
              </div>
              {u.preview ? (
                <p className="mt-1 line-clamp-2 text-[10px] leading-relaxed text-white/50">{u.preview}</p>
              ) : null}
              <div className="mt-2 flex flex-wrap gap-1.5">
                <button
                  type="button"
                  disabled={u.status !== "ready" && u.status !== "imported"}
                  onClick={() => onAnalyze(u.id, "Summarize this resume")}
                  className="rounded-lg bg-white/15 px-2 py-1 text-[10px] font-semibold text-white disabled:opacity-40"
                >
                  Analyze
                </button>
                <button
                  type="button"
                  disabled={u.status !== "ready" && u.status !== "imported"}
                  onClick={() => onAnalyze(u.id, "Extract skills from the uploaded resume")}
                  className="rounded-lg bg-white/10 px-2 py-1 text-[10px] text-white/90 disabled:opacity-40"
                >
                  Extract skills
                </button>
                {u.status === "ready" ? (
                  <button
                    type="button"
                    onClick={async () => {
                      await importCopilotUpload(u.id);
                      onChange();
                    }}
                    className="rounded-lg bg-emerald-400/20 px-2 py-1 text-[10px] font-medium text-emerald-100"
                  >
                    Import to ATS
                  </button>
                ) : null}
              </div>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}
