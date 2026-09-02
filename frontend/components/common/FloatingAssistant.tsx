"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import ReactMarkdown from "react-markdown";
import { ApiClientError } from "@/lib/api-client";
import {
  assistantSessionId,
  formatAssistantAnswer,
  pageQuickActions,
  sendAssistantChat,
  uploadAssistantAttachment,
} from "@/services/assistant.service";

type Message = {
  role: "assistant" | "user";
  content: string;
  intent?: string;
  confidence?: number;
  source?: string;
  suggestedActions?: string[];
  sourceDocuments?: { label: string; similarity?: number }[];
};

type PendingFile = {
  file: File;
  name: string;
  status: "ready" | "uploading" | "processing" | "attached" | "error" | "unavailable";
  uploadId?: string;
  error?: string;
};

const welcome: Message = {
  role: "assistant",
  content:
    "Hi — I’m your AI Recruiting Assistant. Ask about jobs, candidates, applications, or attach a resume (PDF, DOCX, TXT) to analyze.",
};

function shouldShowMeta(message: Message) {
  const hasIntent = Boolean(message.intent && message.intent !== "GENERAL");
  const hasSource =
    Boolean(message.source) &&
    message.source !== "none" &&
    message.source !== "general_knowledge";
  const hasConfidence =
    typeof message.confidence === "number" &&
    message.confidence > 0 &&
    !(message.intent === "GENERAL" && message.confidence < 80);
  return hasIntent || hasSource || hasConfidence;
}

export default function FloatingAssistant() {
  const pathname = usePathname() || "/dashboard";
  const router = useRouter();
  const [mounted, setMounted] = useState(false);
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState("");
  const [messages, setMessages] = useState<Message[]>([welcome]);
  const [isLoading, setIsLoading] = useState(false);
  const [pending, setPending] = useState<PendingFile | null>(null);
  const timeoutRef = useRef<number | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const fileRef = useRef<HTMLInputElement | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const sessionId = useMemo(() => (typeof window !== "undefined" ? assistantSessionId() : ""), []);
  const chips = useMemo(() => pageQuickActions(pathname), [pathname]);

  function openPanel() {
    if (timeoutRef.current) window.clearTimeout(timeoutRef.current);
    setMounted(true);
    requestAnimationFrame(() => setOpen(true));
  }

  function closePanel() {
    setOpen(false);
    timeoutRef.current = window.setTimeout(() => setMounted(false), 260);
  }

  useEffect(() => {
    return () => {
      if (timeoutRef.current) window.clearTimeout(timeoutRef.current);
    };
  }, []);

  useEffect(() => {
    if (!scrollRef.current) return;
    scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
  }, [messages, open, pending]);

  async function attachFile(file: File) {
    const name = file.name;
    const ext = name.toLowerCase().slice(name.lastIndexOf("."));
    if (![".pdf", ".docx", ".txt"].includes(ext)) {
      setPending({
        file,
        name,
        status: "error",
        error: "Supported formats: PDF, DOCX, TXT",
      });
      return;
    }

    setPending({ file, name, status: "uploading" });
    try {
      const uploaded = await uploadAssistantAttachment(file);
      setPending({
        file,
        name,
        status: "attached",
        uploadId: uploaded.id,
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : "Upload failed";
      setPending({ file, name, status: "error", error: message });
    }
  }

  async function sendMessage(prompt: string) {
    const normalized = prompt.trim();
    if (!normalized || !sessionId) return;

    const uploadId =
      pending?.status === "attached" && pending.uploadId ? pending.uploadId : undefined;

    setDraft("");
    setIsLoading(true);
    setMessages((prev) => [
      ...prev,
      {
        role: "user",
        content: uploadId ? `${normalized}\n\n(Attached: ${pending?.name})` : normalized,
      },
      { role: "assistant", content: "Thinking…" },
    ]);

    try {
      const data = await sendAssistantChat(normalized, sessionId, uploadId);
      const answer = formatAssistantAnswer(data);
      setMessages((prev) => [
        ...prev.slice(0, -1),
        {
          role: "assistant",
          content: answer,
          intent: data.intent,
          confidence: data.confidence,
          source: data.source,
          suggestedActions: Array.isArray(data.suggested_actions) ? data.suggested_actions : [],
          sourceDocuments: Array.isArray(data.source_documents) ? data.source_documents : [],
        },
      ]);
    } catch (err) {
      const message =
        err instanceof ApiClientError
          ? err.message
          : err instanceof Error
            ? err.message
            : "Unable to reach the assistant.";
      setMessages((prev) => [...prev.slice(0, -1), { role: "assistant", content: message }]);
    } finally {
      setIsLoading(false);
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }

  function onSuggestion(prompt: string) {
    void sendMessage(prompt);
  }

  return (
    <div>
      <div className="fixed right-6 bottom-6 z-50 flex items-end">
        <button
          aria-label={open ? "Close AI assistant" : "Open AI assistant"}
          onClick={() => (mounted ? (open ? closePanel() : openPanel()) : openPanel())}
          className="inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-slate-900 to-[var(--brand)] text-white shadow-2xl ring-1 ring-white/15 transition hover:scale-[1.03] focus:outline-none"
          title="AI Recruiting Assistant"
        >
          <svg viewBox="0 0 24 24" className="h-6 w-6" fill="none" stroke="currentColor" strokeWidth="1.7">
            <path strokeLinecap="round" strokeLinejoin="round" d="M8 10h8M8 14h5M7 4h10a3 3 0 013 3v8a3 3 0 01-3 3H11l-4 3v-3H7a3 3 0 01-3-3V7a3 3 0 013-3z" />
          </svg>
        </button>
      </div>

      {mounted ? (
        <div
          className={`fixed right-4 bottom-[5.5rem] z-50 flex h-[min(560px,calc(100vh-7rem))] w-[min(400px,calc(100vw-1.5rem))] flex-col overflow-hidden rounded-[28px] border border-slate-200/80 bg-white/90 shadow-2xl backdrop-blur-xl transition-all duration-250 dark:border-slate-700 dark:bg-slate-950/90 ${
            open ? "translate-y-0 opacity-100" : "pointer-events-none translate-y-4 opacity-0"
          }`}
          role="dialog"
          aria-modal="true"
          aria-label="AI Recruiting Assistant"
        >
          <header className="flex items-center justify-between gap-3 border-b border-white/10 bg-gradient-to-br from-slate-950 via-slate-900 to-[var(--brand)] px-4 py-3.5 text-white">
            <div>
              <p className="text-sm font-semibold tracking-tight">AI Recruiting Assistant</p>
              <p className="text-[11px] text-white/65">Jobs · Candidates · Resume analysis</p>
            </div>
            <button
              aria-label="Close assistant"
              onClick={closePanel}
              className="inline-flex h-9 w-9 items-center justify-center rounded-xl bg-white/10 hover:bg-white/20"
            >
              <svg viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="2">
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </header>

          <div className="flex flex-wrap gap-1.5 border-b border-slate-200/70 bg-slate-50/80 px-3 py-2 dark:border-slate-800 dark:bg-slate-900/50">
            {chips.slice(0, 5).map((chip) => (
              <button
                key={chip.label}
                type="button"
                onClick={() => onSuggestion(chip.prompt)}
                className="rounded-full border border-slate-200 bg-white px-2.5 py-1 text-[11px] font-medium text-slate-700 transition hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200"
              >
                {chip.label}
              </button>
            ))}
          </div>

          <div ref={scrollRef} className="flex-1 space-y-3 overflow-y-auto bg-slate-50/60 px-3 py-3 dark:bg-slate-950/60">
            {messages.slice(-20).map((message, index) => (
              <div key={index} className={`flex ${message.role === "user" ? "justify-end" : "justify-start"}`}>
                <div
                  className={`max-w-[90%] rounded-[20px] px-3.5 py-2.5 text-sm shadow-sm ${
                    message.role === "user"
                      ? "bg-[var(--brand)] text-white"
                      : "border border-slate-200/80 bg-white text-slate-900 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
                  }`}
                >
                  {message.role === "assistant" && shouldShowMeta(message) ? (
                    <div className="mb-1.5 flex flex-wrap gap-1.5">
                      {message.intent && message.intent !== "GENERAL" ? (
                        <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-semibold uppercase text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                          {message.intent}
                        </span>
                      ) : null}
                      {message.source && message.source !== "none" && message.source !== "general_knowledge" ? (
                        <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                          {message.source}
                        </span>
                      ) : null}
                      {typeof message.confidence === "number" &&
                      message.confidence > 0 &&
                      !(message.intent === "GENERAL" && message.confidence < 80) ? (
                        <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-medium text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300">
                          {Math.round(message.confidence)}% confidence
                        </span>
                      ) : null}
                    </div>
                  ) : null}

                  {message.role === "assistant" ? (
                    <div className="prose prose-sm max-w-none dark:prose-invert prose-p:my-1 prose-ul:my-1">
                      <ReactMarkdown>{message.content}</ReactMarkdown>
                    </div>
                  ) : (
                    <p className="whitespace-pre-wrap">{message.content}</p>
                  )}

                  {message.sourceDocuments && message.sourceDocuments.length > 0 ? (
                    <details className="mt-2 text-[11px] text-slate-500">
                      <summary className="cursor-pointer font-medium">Sources</summary>
                      <ul className="mt-1 space-y-0.5">
                        {message.sourceDocuments.slice(0, 5).map((doc, i) => (
                          <li key={`${doc.label}-${i}`}>
                            • {doc.label}
                            {doc.similarity != null ? ` (${Math.round(doc.similarity)}%)` : ""}
                          </li>
                        ))}
                      </ul>
                    </details>
                  ) : null}

                  {message.suggestedActions && message.suggestedActions.length > 0 ? (
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {message.suggestedActions.slice(0, 3).map((action) => (
                        <button
                          key={action}
                          type="button"
                          onClick={() => {
                            if (action.startsWith("/")) router.push(action);
                            else onSuggestion(action);
                          }}
                          className="rounded-lg border border-slate-200 bg-slate-50 px-2 py-1 text-[11px] font-medium text-slate-700 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
                        >
                          {action}
                        </button>
                      ))}
                    </div>
                  ) : null}
                </div>
              </div>
            ))}
          </div>

          <div className="border-t border-slate-200/80 bg-white/95 p-3 dark:border-slate-800 dark:bg-slate-950/95">
            {pending ? (
              <div className="mb-2 flex items-center gap-2 rounded-2xl border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-900">
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-semibold text-slate-800 dark:text-slate-100">{pending.name}</p>
                  <p className="text-[10px] text-slate-500">
                    {pending.status === "uploading" && "Uploading…"}
                    {pending.status === "processing" && "Processing…"}
                    {pending.status === "attached" && "Attached · ready to analyze"}
                    {pending.status === "unavailable" && (pending.error || "Upload unavailable")}
                    {pending.status === "error" && (pending.error || "Failed")}
                    {pending.status === "ready" && "Ready"}
                  </p>
                </div>
                <button
                  type="button"
                  aria-label="Remove attachment"
                  onClick={() => setPending(null)}
                  className="inline-flex h-7 w-7 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-200/70 dark:hover:bg-slate-800"
                >
                  <svg viewBox="0 0 24 24" className="h-3.5 w-3.5" fill="none" stroke="currentColor" strokeWidth="2">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ) : null}

            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (!draft.trim() || isLoading) return;
                void sendMessage(draft);
              }}
              className="flex items-end gap-2"
            >
              <input
                ref={fileRef}
                type="file"
                accept=".pdf,.docx,.txt,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) void attachFile(file);
                  e.target.value = "";
                }}
              />
              <button
                type="button"
                aria-label="Attach file"
                title="Attach PDF, DOCX, or TXT"
                onClick={() => fileRef.current?.click()}
                disabled={isLoading}
                className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-slate-200 bg-slate-50 text-slate-600 transition hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300"
              >
                <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="1.8">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M21 12.5V8a5 5 0 00-10 0v9a3 3 0 006 0V9"
                  />
                </svg>
              </button>
              <input
                ref={inputRef}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="Ask a question…"
                disabled={isLoading}
                className="flex-1 rounded-2xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-[var(--brand)] focus:ring-2 focus:ring-[var(--brand)]/15 disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
              />
              <button
                type="submit"
                disabled={!draft.trim() || isLoading}
                className="inline-flex h-11 items-center justify-center rounded-2xl bg-[var(--brand)] px-4 text-sm font-semibold text-white transition hover:bg-[var(--brand-hover)] disabled:cursor-not-allowed disabled:bg-slate-400"
              >
                {isLoading ? "…" : "Send"}
              </button>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  );
}
