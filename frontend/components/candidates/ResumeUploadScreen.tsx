"use client";

import { useCallback, useRef, useState } from "react";
import { Button } from "@/components/ui/Button";
import { ApiClientError } from "@/lib/api-client";
import type { ResumeUploadResult } from "@/lib/resume-types";
import { uploadResume } from "@/services/resume.service";

const ACCEPTED_EXTENSIONS = [".pdf", ".docx"];
const MAX_FILE_SIZE_MB = 5;

type UploadPhase = "idle" | "uploading" | "parsing" | "done" | "error";

interface ResumeUploadScreenProps {
  onUploadComplete: (result: ResumeUploadResult) => void;
  onCancel?: () => void;
}

function isAcceptedFile(file: File) {
  const lower = file.name.toLowerCase();
  return ACCEPTED_EXTENSIONS.some((ext) => lower.endsWith(ext));
}

export function ResumeUploadScreen({ onUploadComplete, onCancel }: ResumeUploadScreenProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [phase, setPhase] = useState<UploadPhase>("idle");
  const [progress, setProgress] = useState(0);
  const [fileName, setFileName] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [lastFile, setLastFile] = useState<File | null>(null);

  const resetInput = () => {
    if (inputRef.current) inputRef.current.value = "";
  };

  const runUpload = useCallback(
    async (file: File) => {
      if (!isAcceptedFile(file)) {
        setError("Please upload a PDF or DOCX file.");
        setPhase("error");
        resetInput();
        return;
      }

      if (file.size > MAX_FILE_SIZE_MB * 1024 * 1024) {
        setError(`File must be ${MAX_FILE_SIZE_MB}MB or smaller.`);
        setPhase("error");
        resetInput();
        return;
      }

      setError(null);
      setLastFile(file);
      setFileName(file.name);
      setPhase("uploading");
      setProgress(15);

      const progressTimer = window.setInterval(() => {
        setProgress((p) => (p < 60 ? p + 4 : p));
      }, 120);

      try {
        // Server uploads and parses in one request.
        setPhase("uploading");
        const result = await uploadResume(file);
        window.clearInterval(progressTimer);
        setPhase("parsing");
        setProgress(90);

        // Reflect server parsing outcome in UI states.
        await new Promise((r) => setTimeout(r, 250));
        setProgress(100);
        setPhase("done");
        onUploadComplete(result);
      } catch (err) {
        window.clearInterval(progressTimer);
        setPhase("error");
        setProgress(0);
        setError(err instanceof ApiClientError ? err.message : "Upload failed. Please try again.");
        resetInput();
      }
    },
    [onUploadComplete],
  );

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (file) void runUpload(file);
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(false);
    if (phase === "uploading" || phase === "parsing") return;
    const file = e.dataTransfer.files?.[0];
    if (file) void runUpload(file);
  }

  const busy = phase === "uploading" || phase === "parsing";

  return (
    <div className="mx-auto max-w-xl">
      <div
        role="button"
        tabIndex={0}
        aria-label="Upload resume file"
        onKeyDown={(e) => {
          if (busy) return;
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            inputRef.current?.click();
          }
        }}
        onDragOver={(e) => {
          e.preventDefault();
          if (!busy) setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
        onClick={() => {
          if (!busy) inputRef.current?.click();
        }}
        className={`rounded-2xl border-2 border-dashed bg-white p-10 text-center shadow-sm transition ${
          dragOver
            ? "border-blue-400 bg-blue-50/40 ring-2 ring-blue-100"
            : "border-slate-200 ring-1 ring-slate-200 hover:border-blue-300 hover:bg-slate-50/60"
        } ${busy ? "pointer-events-none opacity-90" : "cursor-pointer"}`}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
          className="hidden"
          disabled={busy}
          onChange={handleFileChange}
        />

        {busy ? (
          <div className="space-y-4">
            <div className="mx-auto h-10 w-10 animate-spin rounded-full border-2 border-slate-200 border-t-blue-600" />
            <p className="text-base font-semibold text-slate-900">
              {phase === "parsing" ? "Parsing Resume..." : "Uploading Resume..."}
            </p>
            {fileName && <p className="truncate text-sm text-slate-500">{fileName}</p>}
            <div className="ats-progress mx-auto max-w-sm" role="progressbar" aria-valuenow={progress} aria-valuemin={0} aria-valuemax={100}>
              <span style={{ width: `${progress}%` }} />
            </div>
            <p className="text-sm font-medium text-blue-600">{progress}%</p>
            <p className="text-xs text-slate-500">
              {phase === "parsing"
                ? "Extracting name, skills, experience, education, and more…"
                : "Securely uploading your file…"}
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-blue-50 text-blue-600 ring-1 ring-blue-100">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.75"
                className="h-7 w-7"
                aria-hidden
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5"
                />
              </svg>
            </div>
            <p className="text-base font-semibold text-slate-900">Upload a resume</p>
            <p className="text-sm text-slate-500">
              Drag and drop a file here, or click to browse.
              <br />
              PDF or DOCX · up to {MAX_FILE_SIZE_MB}MB
            </p>
            <Button
              type="button"
              className="mx-auto mt-2 w-auto px-6"
              onClick={(e) => {
                e.stopPropagation();
                inputRef.current?.click();
              }}
            >
              Choose file
            </Button>
          </div>
        )}
      </div>

      {error && (
        <div className="mt-4 space-y-3 text-center" role="alert">
          <p className="rounded-xl bg-red-50 px-4 py-3 text-sm font-medium text-red-700 ring-1 ring-red-100">
            {error}
          </p>
          {lastFile && (
            <Button
              type="button"
              variant="secondary"
              className="mx-auto w-auto px-5"
              onClick={() => void runUpload(lastFile)}
            >
              Try again
            </Button>
          )}
        </div>
      )}

      {onCancel && !busy && (
        <div className="mt-6 flex justify-center">
          <Button type="button" variant="secondary" className="w-auto px-5" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      )}
    </div>
  );
}
