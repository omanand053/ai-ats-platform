"use client";

import ReactMarkdown from "react-markdown";
import type { CopilotPayload, SourceDoc, UICard, QuickAction } from "@/services/copilot.service";

export function ConfidenceBadge({ value }: { value?: number }) {
  if (value == null) return null;
  const v = Math.round(value);
  const tone =
    v >= 80 ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300" :
    v >= 50 ? "bg-amber-500/15 text-amber-700 dark:text-amber-300" :
    "bg-rose-500/15 text-rose-700 dark:text-rose-300";
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-semibold ${tone}`}>
      {v}% confidence
    </span>
  );
}

export function IntentBadge({ intent, source }: { intent?: string; source?: string }) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {intent ? (
        <span className="rounded-full bg-slate-200/70 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-600 dark:bg-slate-700/80 dark:text-slate-200">
          {intent}
        </span>
      ) : null}
      {source ? (
        <span className="rounded-full bg-slate-200/70 px-2 py-0.5 text-[10px] font-medium text-slate-600 dark:bg-slate-700/80 dark:text-slate-200">
          {source}
        </span>
      ) : null}
    </div>
  );
}

export function MarkdownAnswer({ content }: { content: string }) {
  return (
    <div className="prose prose-sm max-w-none dark:prose-invert prose-p:my-1.5 prose-ul:my-1.5 prose-li:my-0.5 prose-headings:my-2 prose-pre:bg-slate-900/90 prose-pre:text-slate-100">
      <ReactMarkdown
        components={{
          table: ({ children }) => (
            <div className="my-2 overflow-x-auto rounded-xl border border-slate-200 dark:border-slate-700">
              <table className="min-w-full text-left text-xs">{children}</table>
            </div>
          ),
          th: ({ children }) => <th className="bg-slate-100 px-2 py-1.5 font-semibold dark:bg-slate-800">{children}</th>,
          td: ({ children }) => <td className="border-t border-slate-100 px-2 py-1.5 dark:border-slate-800">{children}</td>,
          a: ({ href, children }) => (
            <a href={href} className="text-[var(--brand)] underline-offset-2 hover:underline" target="_blank" rel="noreferrer">
              {children}
            </a>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}

export function SourcePanel({ docs, count }: { docs?: SourceDoc[]; count?: number }) {
  if (!docs?.length) return null;
  return (
    <details className="mt-2 rounded-xl border border-slate-200/80 bg-white/50 open:bg-white/80 dark:border-slate-700 dark:bg-slate-900/40">
      <summary className="cursor-pointer list-none px-3 py-2 text-[11px] font-semibold text-slate-600 dark:text-slate-300">
        Sources {count != null ? `(${count})` : `(${docs.length})`}
      </summary>
      <ul className="space-y-1 border-t border-slate-100 px-3 py-2 text-[11px] text-slate-500 dark:border-slate-800 dark:text-slate-400">
        {docs.slice(0, 8).map((doc, i) => (
          <li key={`${doc.id || doc.label}-${i}`}>
            • {doc.label}
            {doc.similarity != null ? ` · ${Math.round(doc.similarity)}%` : ""}
          </li>
        ))}
      </ul>
    </details>
  );
}

export function CardGrid({ cards }: { cards?: UICard[] }) {
  if (!cards?.length) return null;
  return (
    <div className="mt-2 grid gap-2">
      {cards.slice(0, 6).map((card, i) => (
        <a
          key={`${card.type}-${card.title}-${i}`}
          href={card.href || "#"}
          className="block rounded-2xl border border-slate-200/80 bg-gradient-to-br from-white to-slate-50 px-3 py-2 text-left shadow-sm transition hover:border-slate-300 dark:border-slate-700 dark:from-slate-900 dark:to-slate-950"
        >
          <p className="text-[10px] font-semibold uppercase tracking-wide text-slate-400">{card.type}</p>
          <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{card.title}</p>
          {card.subtitle ? <p className="text-xs text-slate-500">{card.subtitle}</p> : null}
        </a>
      ))}
    </div>
  );
}

export function FunnelMiniChart({ data }: { data?: Record<string, unknown> }) {
  const funnel = data?.funnel;
  if (!Array.isArray(funnel) || funnel.length === 0) return null;
  const rows = funnel as { name?: string; Name?: string; count?: number; Count?: number }[];
  const max = Math.max(...rows.map((r) => Number(r.count ?? r.Count ?? 0)), 1);
  return (
    <div className="mt-2 space-y-1.5 rounded-2xl border border-slate-200/70 bg-white/60 p-3 dark:border-slate-700 dark:bg-slate-900/50">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-slate-400">Hiring funnel</p>
      {rows.map((r, i) => {
        const name = r.name ?? r.Name ?? `Stage ${i + 1}`;
        const count = Number(r.count ?? r.Count ?? 0);
        return (
          <div key={name} className="flex items-center gap-2 text-[11px]">
            <span className="w-28 truncate text-slate-600 dark:text-slate-300">{name}</span>
            <div className="h-2 flex-1 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
              <div
                className="h-full rounded-full bg-gradient-to-r from-[var(--brand)] to-indigo-400"
                style={{ width: `${Math.max(6, (count / max) * 100)}%` }}
              />
            </div>
            <span className="w-8 text-right font-semibold text-slate-700 dark:text-slate-200">{count}</span>
          </div>
        );
      })}
    </div>
  );
}

export function ActionRow({
  actions,
  onAction,
}: {
  actions?: QuickAction[];
  onAction: (a: QuickAction) => void;
}) {
  if (!actions?.length) return null;
  return (
    <div className="mt-2 flex flex-wrap gap-1.5">
      {actions.slice(0, 4).map((a) => (
        <button
          key={a.id || a.label}
          type="button"
          onClick={() => onAction(a)}
          className="rounded-xl border border-slate-200/90 bg-white/80 px-2.5 py-1 text-[11px] font-medium text-slate-700 transition hover:border-slate-300 hover:bg-white dark:border-slate-600 dark:bg-slate-900/70 dark:text-slate-200"
        >
          {a.label}
        </button>
      ))}
    </div>
  );
}

export function TypingDots() {
  return (
    <span className="inline-flex items-center gap-1 px-1">
      <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-slate-400 [animation-delay:-0.2s]" />
      <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-slate-400 [animation-delay:-0.1s]" />
      <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-slate-400" />
    </span>
  );
}

export function enrichMessage(data: CopilotPayload) {
  return {
    content: (data.answer || data.reply || "").trim() || "No response.",
    intent: data.intent,
    confidence: data.confidence,
    source: data.source,
    suggestedActions: data.suggested_actions ?? [],
    quickActions: data.quick_actions ?? [],
    sourceDocuments: data.source_documents ?? [],
    retrievedContextCount: data.retrieved_context_count,
    cards: data.cards ?? [],
    data: data.data,
  };
}
