"use client";

import { useEffect, useMemo, useState } from "react";
import { CandidateStatusBadge } from "@/components/candidates/candidate-utils";
import { AICopilotPanel } from "@/components/jobs/recruiter/AICopilotPanel";
import {
  confidenceTone,
  formatPct,
  type RankedApplicant,
} from "@/components/jobs/recruiter/filters";
import { Avatar } from "@/components/ui/Avatar";
import { Button } from "@/components/ui/Button";
import { Drawer } from "@/components/ui/Drawer";
import { decodeResumeExtras } from "@/lib/candidate-validators";
import { ApiClientError } from "@/lib/api-client";
import { createResumeObjectUrl, viewResumeFile } from "@/lib/resume-file";
import type { Job } from "@/lib/job-types";
import {
  createCandidateNote,
  deleteCandidateNote,
  getCandidateTimeline,
  listCandidateNotes,
  type CandidateNote,
  type CandidateTimelineItem,
} from "@/services/candidate.service";
import {
  createCandidateComment,
  listCandidateComments,
  type CollaborationComment,
} from "@/services/enterprise.service";

function highlightText(text: string, matched: string[], missing: string[]) {
  if (!text) return "—";
  const terms = [
    ...matched.map((s) => ({ term: s, tone: "match" as const })),
    ...missing.map((s) => ({ term: s, tone: "miss" as const })),
  ].filter((t) => t.term.trim().length > 1);
  if (!terms.length) return text;

  const escaped = terms
    .map((t) => t.term.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
    .sort((a, b) => b.length - a.length);
  const re = new RegExp(`(${escaped.join("|")})`, "gi");
  const parts = text.split(re);
  const lookup = new Map(terms.map((t) => [t.term.toLowerCase(), t.tone]));

  return parts.map((part, i) => {
    const tone = lookup.get(part.toLowerCase());
    if (!tone) return <span key={i}>{part}</span>;
    return (
      <mark
        key={i}
        className={
          tone === "match"
            ? "rounded bg-emerald-100 px-0.5 text-emerald-900"
            : "rounded bg-amber-100 px-0.5 text-amber-900"
        }
      >
        {part}
      </mark>
    );
  });
}

export function CandidateDrawer({
  open,
  row,
  job,
  comparePool = [],
  onClose,
  onToast,
}: {
  open: boolean;
  row: RankedApplicant | null;
  job: Job;
  comparePool?: RankedApplicant[];
  onClose: () => void;
  onToast: (message: string, tone?: "success" | "error") => void;
}) {
  const candidate = row?.candidate;
  const match = row?.match;
  const [drawerTab, setDrawerTab] = useState<"profile" | "copilot">("profile");
  const [notes, setNotes] = useState<CandidateNote[]>([]);
  const [timeline, setTimeline] = useState<CandidateTimelineItem[]>([]);
  const [comments, setComments] = useState<CollaborationComment[]>([]);
  const [noteBody, setNoteBody] = useState("");
  const [commentBody, setCommentBody] = useState("");
  const [loadingMeta, setLoadingMeta] = useState(false);
  const [savingNote, setSavingNote] = useState(false);
  const [savingComment, setSavingComment] = useState(false);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewMime, setPreviewMime] = useState("");

  const extras = useMemo(
    () => decodeResumeExtras(candidate?.resume_text),
    [candidate?.resume_text],
  );

  useEffect(() => {
    if (!open) return;
    setDrawerTab("profile");
  }, [open, candidate?.id]);

  useEffect(() => {
    if (!open || !candidate) return;
    let cancelled = false;
    setLoadingMeta(true);
    Promise.allSettled([
      listCandidateNotes(candidate.id),
      getCandidateTimeline(candidate.id),
      listCandidateComments(candidate.id),
    ]).then(([notesRes, timelineRes, commentsRes]) => {
      if (cancelled) return;
      if (notesRes.status === "fulfilled") setNotes(notesRes.value);
      else setNotes([]);
      if (timelineRes.status === "fulfilled") setTimeline(timelineRes.value);
      else setTimeline([]);
      if (commentsRes.status === "fulfilled") setComments(commentsRes.value);
      else setComments([]);
      setLoadingMeta(false);
    });
    return () => {
      cancelled = true;
    };
  }, [open, candidate?.id]);

  useEffect(() => {
    if (!open || !candidate?.resume_url) {
      setPreviewUrl(null);
      return;
    }
    let revoked: string | null = null;
    let cancelled = false;
    createResumeObjectUrl(candidate.resume_url)
      .then(({ objectUrl, mime }) => {
        if (cancelled) {
          URL.revokeObjectURL(objectUrl);
          return;
        }
        revoked = objectUrl;
        setPreviewUrl(objectUrl);
        setPreviewMime(mime);
      })
      .catch(() => {
        if (!cancelled) setPreviewUrl(null);
      });
    return () => {
      cancelled = true;
      if (revoked) URL.revokeObjectURL(revoked);
    };
  }, [open, candidate?.resume_url]);

  async function handleAddNote() {
    if (!candidate || !noteBody.trim() || savingNote) return;
    setSavingNote(true);
    try {
      const note = await createCandidateNote(candidate.id, noteBody.trim());
      setNotes((prev) => [note, ...prev]);
      setNoteBody("");
      onToast("Note saved");
      const refreshed = await getCandidateTimeline(candidate.id).catch(() => null);
      if (refreshed) setTimeline(refreshed);
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to save note";
      onToast(message, "error");
    } finally {
      setSavingNote(false);
    }
  }

  async function handleAddComment() {
    if (!candidate || !commentBody.trim() || savingComment) return;
    setSavingComment(true);
    try {
      const comment = await createCandidateComment(candidate.id, commentBody.trim());
      setComments((prev) => [comment, ...prev]);
      setCommentBody("");
      onToast("Comment posted");
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to post comment";
      onToast(message, "error");
    } finally {
      setSavingComment(false);
    }
  }

  async function handleDeleteNote(noteId: string) {
    if (!candidate) return;
    try {
      await deleteCandidateNote(candidate.id, noteId);
      setNotes((prev) => prev.filter((n) => n.id !== noteId));
      onToast("Note deleted");
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to delete note";
      onToast(message, "error");
    }
  }

  if (!candidate) {
    return <Drawer open={open} onClose={onClose} title="Candidate" />;
  }

  const matched = match?.matched_skills ?? candidate.matched_skills ?? [];
  const missing = match?.missing_skills ?? candidate.missing_skills ?? [];

  return (
    <Drawer open={open} onClose={onClose} title={candidate.name} widthClass="max-w-2xl">
      <div className="space-y-6">
        <div className="flex flex-wrap items-start gap-3">
          <Avatar name={candidate.name} size="lg" />
          <div className="min-w-0 flex-1">
            <p className="text-sm text-slate-600">
              {candidate.current_designation || "No designation"}
              {candidate.current_company ? ` @ ${candidate.current_company}` : ""}
            </p>
            <div className="mt-2 flex flex-wrap gap-2">
              <span className="rounded-lg bg-blue-50 px-2.5 py-1 text-xs font-bold text-blue-800 ring-1 ring-blue-100">
                AI Match {formatPct(match?.ai_match_score)}
              </span>
              <span
                className={`rounded-lg px-2.5 py-1 text-xs font-semibold capitalize ring-1 ${confidenceTone(match?.confidence)}`}
              >
                {match?.confidence || "—"} confidence
              </span>
              <span className="rounded-lg bg-slate-50 px-2.5 py-1 text-xs font-semibold text-slate-700 ring-1 ring-slate-200">
                Semantic {formatPct(match?.similarity_score)}
              </span>
              <CandidateStatusBadge status={candidate.status} />
            </div>
          </div>
        </div>

        <div className="flex gap-2 border-b border-slate-200 pb-2">
          <button
            type="button"
            className={`rounded-lg px-3 py-1.5 text-xs font-semibold ${
              drawerTab === "profile" ? "bg-slate-900 text-white" : "bg-slate-100 text-slate-600"
            }`}
            onClick={() => setDrawerTab("profile")}
          >
            Profile
          </button>
          <button
            type="button"
            className={`rounded-lg px-3 py-1.5 text-xs font-semibold ${
              drawerTab === "copilot" ? "bg-slate-900 text-white" : "bg-slate-100 text-slate-600"
            }`}
            onClick={() => setDrawerTab("copilot")}
          >
            AI Copilot
          </button>
        </div>

        {drawerTab === "copilot" ? (
          <AICopilotPanel
            jobId={job.id}
            candidateId={candidate.id}
            comparePool={comparePool}
            onToast={onToast}
          />
        ) : (
          <>
        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">AI Summary</h3>
          <p className="mt-1 text-sm leading-6 text-slate-700">
            {candidate.resume_summary ||
              match?.why_shortlisted ||
              "No AI summary available yet."}
          </p>
        </section>

        <section className="grid gap-4 sm:grid-cols-2">
          <div>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Strengths</h3>
            <ul className="mt-1 list-disc space-y-1 pl-5 text-sm text-slate-700">
              {(match?.strengths?.length ? match.strengths : ["—"]).map((s) => (
                <li key={s}>{s}</li>
              ))}
            </ul>
          </div>
          <div>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Missing Skills
            </h3>
            <div className="mt-1 flex flex-wrap gap-1.5">
              {missing.length ? (
                missing.map((s) => (
                  <span
                    key={s}
                    className="rounded-lg bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-900 ring-1 ring-amber-100"
                  >
                    {s}
                  </span>
                ))
              ) : (
                <span className="text-sm text-slate-500">None highlighted</span>
              )}
            </div>
          </div>
        </section>

        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Skills</h3>
          <div className="mt-1 flex flex-wrap gap-1.5">
            {(candidate.skills ?? []).map((s) => (
              <span
                key={s}
                className={`rounded-lg px-2 py-0.5 text-xs font-medium ring-1 ${
                  matched.some((m) => m.toLowerCase() === s.toLowerCase())
                    ? "bg-emerald-50 text-emerald-800 ring-emerald-100"
                    : "bg-slate-50 text-slate-700 ring-slate-200"
                }`}
              >
                {s}
              </span>
            ))}
            {!candidate.skills?.length && <span className="text-sm text-slate-500">—</span>}
          </div>
        </section>

        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Experience</h3>
          <p className="mt-1 text-sm text-slate-700">
            {typeof candidate.experience_years === "number"
              ? `${candidate.experience_years} years`
              : "—"}
            {job.experience_required ? ` · Role requires ${job.experience_required}` : ""}
          </p>
        </section>

        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Education</h3>
          {extras.education.length ? (
            <ul className="mt-1 space-y-1 text-sm text-slate-700">
              {extras.education.map((e, i) => (
                <li key={i}>
                  {[e.degree, e.branch, e.school, e.years].filter(Boolean).join(" · ") || "Education entry"}
                </li>
              ))}
            </ul>
          ) : (
            <p className="mt-1 text-sm text-slate-500">No education parsed</p>
          )}
        </section>

        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Projects</h3>
          {extras.projects.length ? (
            <ul className="mt-1 space-y-2 text-sm text-slate-700">
              {extras.projects.map((p, i) => (
                <li key={i} className="rounded-xl bg-slate-50 px-3 py-2 ring-1 ring-slate-100">
                  <p className="font-medium">{p.name}</p>
                  {p.technologies?.length ? (
                    <p className="mt-0.5 text-xs text-slate-500">{p.technologies.join(", ")}</p>
                  ) : null}
                  {p.description ? <p className="mt-1 text-slate-600">{p.description}</p> : null}
                </li>
              ))}
            </ul>
          ) : (
            <p className="mt-1 text-sm text-slate-500">No projects parsed</p>
          )}
        </section>

        <section>
          <div className="flex items-center justify-between gap-2">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Resume Preview
            </h3>
            {candidate.resume_url && (
              <Button
                size="sm"
                variant="secondary"
                className="w-auto px-3"
                onClick={() => viewResumeFile(candidate.resume_url!).catch(() => onToast("Unable to open resume", "error"))}
              >
                Open file
              </Button>
            )}
          </div>
          {previewUrl && previewMime.includes("pdf") ? (
            <iframe
              title="Resume PDF"
              src={previewUrl}
              className="mt-2 h-72 w-full rounded-xl border border-slate-200"
            />
          ) : (
            <pre className="mt-2 max-h-72 overflow-auto whitespace-pre-wrap rounded-xl bg-slate-50 p-3 text-xs leading-5 text-slate-700 ring-1 ring-slate-100">
              {highlightText(extras.rawText || candidate.resume_text || "No resume text", matched, missing)}
            </pre>
          )}
        </section>

        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Timeline</h3>
          {loadingMeta ? (
            <p className="mt-2 text-sm text-slate-500">Loading timeline…</p>
          ) : timeline.length ? (
            <ol className="mt-3 space-y-0">
              {timeline.map((item, idx) => (
                <li key={item.id} className="relative flex gap-3 pb-4 last:pb-0">
                  {idx < timeline.length - 1 && (
                    <span className="absolute left-[7px] top-4 h-[calc(100%-8px)] w-px bg-slate-200" />
                  )}
                  <span className="relative mt-1 h-3.5 w-3.5 shrink-0 rounded-full bg-blue-600 ring-4 ring-blue-50" />
                  <div>
                    <p className="text-sm font-medium text-slate-900">{item.label}</p>
                    <p className="text-xs text-slate-500">
                      {new Date(item.timestamp).toLocaleString()}
                      {item.source === "inferred" ? " · inferred" : ""}
                    </p>
                  </div>
                </li>
              ))}
            </ol>
          ) : (
            <p className="mt-2 text-sm text-slate-500">No timeline events yet</p>
          )}
        </section>

        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Recruiter Notes
          </h3>
          <div className="mt-2 space-y-2">
            <textarea
              value={noteBody}
              onChange={(e) => setNoteBody(e.target.value)}
              rows={3}
              placeholder="e.g. Strong communication · Needs system design round · Suitable for L2"
              className="ats-input"
            />
            <Button
              className="w-auto px-4"
              loading={savingNote}
              disabled={!noteBody.trim()}
              onClick={handleAddNote}
            >
              Save note
            </Button>
          </div>
          <ul className="mt-3 space-y-2">
            {notes.map((note) => (
              <li
                key={note.id}
                className="rounded-xl bg-slate-50 px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-100"
              >
                <div className="flex items-start justify-between gap-2">
                  <p className="whitespace-pre-wrap">{note.body}</p>
                  <button
                    type="button"
                    className="text-xs text-rose-600 hover:underline"
                    onClick={() => handleDeleteNote(note.id)}
                  >
                    Delete
                  </button>
                </div>
                <p className="mt-1 text-[11px] text-slate-500">
                  {new Date(note.created_at).toLocaleString()}
                </p>
              </li>
            ))}
            {!notes.length && !loadingMeta && (
              <li className="text-sm text-slate-500">No notes yet</li>
            )}
          </ul>
        </section>

        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Team Collaboration
          </h3>
          <p className="mt-1 text-xs text-slate-500">
            Comments notify teammates. Mentions: include user IDs in the API payload when wiring @-picker.
          </p>
          <div className="mt-2 space-y-2">
            <textarea
              value={commentBody}
              onChange={(e) => setCommentBody(e.target.value)}
              rows={3}
              placeholder="@hiring_manager Can we schedule a panel for Thursday?"
              className="ats-input"
            />
            <Button
              className="w-auto px-4"
              loading={savingComment}
              disabled={!commentBody.trim()}
              onClick={() => void handleAddComment()}
            >
              Post comment
            </Button>
          </div>
          <ul className="mt-3 space-y-2">
            {comments.map((c) => (
              <li
                key={c.id}
                className="rounded-xl bg-slate-50 px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-100 dark:bg-slate-900 dark:ring-slate-800"
              >
                <p className="whitespace-pre-wrap">{c.body}</p>
                <p className="mt-1 text-[11px] text-slate-500">
                  {new Date(c.created_at).toLocaleString()}
                  {c.mentions?.length ? ` · ${c.mentions.length} mention(s)` : ""}
                </p>
              </li>
            ))}
            {!comments.length && !loadingMeta && (
              <li className="text-sm text-slate-500">No team comments yet</li>
            )}
          </ul>
        </section>
          </>
        )}
      </div>
    </Drawer>
  );
}
