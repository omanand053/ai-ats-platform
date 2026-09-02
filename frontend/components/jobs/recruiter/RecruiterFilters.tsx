"use client";

import type { CandidateStatus } from "@/lib/candidate-types";
import { Button } from "@/components/ui/Button";
import {
  defaultFilters,
  type WorkspaceFilters,
} from "@/components/jobs/recruiter/filters";

const STAGES: Array<CandidateStatus | "all"> = [
  "all",
  "applied",
  "screening",
  "ai_shortlisted",
  "recruiter_shortlisted",
  "interview",
  "selected",
  "rejected",
];

export function RecruiterFilters({
  filters,
  onChange,
  onSave,
  onReset,
  resultCount,
}: {
  filters: WorkspaceFilters;
  onChange: (next: WorkspaceFilters) => void;
  onSave: () => void;
  onReset: () => void;
  resultCount: number;
}) {
  return (
    <div className="rounded-[12px] border border-slate-200 bg-white p-2">
      <div className="flex items-center justify-between gap-2">
        <div>
          <p className="text-sm font-semibold text-[#0b1220]">Filters</p>
          <p className="text-xs text-[#6b7280]">{resultCount} applicants match</p>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="secondary" className="w-auto px-2 py-1" onClick={onSave}>
            Save
          </Button>
          <Button
            size="sm"
            variant="secondary"
            className="w-auto px-2 py-1"
            onClick={() => {
              onChange(defaultFilters());
              onReset();
            }}
          >
            Reset
          </Button>
        </div>
      </div>

      <div className="mt-2 grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
        <label className="block text-xs font-medium text-slate-600">
          Search
          <input
            className="ats-input mt-1"
            value={filters.search}
            onChange={(e) => onChange({ ...filters, search: e.target.value })}
            placeholder="Name, email, skills…"
          />
        </label>
        <label className="block text-xs font-medium text-[#6b7280]">
          Stage
          <select
            className="ats-input mt-1"
            value={filters.stage}
            onChange={(e) =>
              onChange({ ...filters, stage: e.target.value as WorkspaceFilters["stage"] })
            }
          >
            {STAGES.map((s) => (
              <option key={s} value={s}>
                {s === "all" ? "All stages" : s.replace(/_/g, " ")}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-xs font-medium text-[#6b7280]">
          Min AI Match ({filters.minAiMatch}%)
          <input
            type="range"
            min={0}
            max={100}
            step={5}
            className="mt-2 w-full"
            value={filters.minAiMatch}
            onChange={(e) => onChange({ ...filters, minAiMatch: Number(e.target.value) })}
          />
        </label>
        <label className="block text-xs font-medium text-[#6b7280]">
          Min experience (yrs)
          <input
            type="number"
            min={0}
            max={40}
            className="ats-input mt-1"
            value={filters.minExperience}
            onChange={(e) =>
              onChange({ ...filters, minExperience: Math.max(0, Number(e.target.value) || 0) })
            }
          />
        </label>
        <label className="block text-xs font-medium text-[#6b7280]">
          Skills
          <input
            className="ats-input mt-1"
            value={filters.skill}
            onChange={(e) => onChange({ ...filters, skill: e.target.value })}
            placeholder="e.g. React"
          />
        </label>
        <label className="block text-xs font-medium text-[#6b7280]">
          Location
          <input
            className="ats-input mt-1"
            value={filters.location}
            onChange={(e) => onChange({ ...filters, location: e.target.value })}
            placeholder="City / remote"
          />
        </label>
        <label className="block text-xs font-medium text-[#6b7280]">
          Confidence
          <select
            className="ats-input mt-1"
            value={filters.confidence}
            onChange={(e) =>
              onChange({
                ...filters,
                confidence: e.target.value as WorkspaceFilters["confidence"],
              })
            }
          >
            <option value="all">All</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
          </select>
        </label>
        <label className="block text-xs font-medium text-slate-600">
          Resume uploaded
          <select
            className="ats-input mt-1"
            value={filters.resumeUploaded}
            onChange={(e) =>
              onChange({
                ...filters,
                resumeUploaded: e.target.value as WorkspaceFilters["resumeUploaded"],
              })
            }
          >
            <option value="all">All</option>
            <option value="yes">Has resume</option>
            <option value="no">Missing resume</option>
          </select>
        </label>
      </div>
    </div>
  );
}
