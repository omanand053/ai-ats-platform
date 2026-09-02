"use client";

import { FormEvent, useEffect, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import {
  CANDIDATE_STATUSES,
  PROCESSING_STATUSES,
  type CandidateFormValues,
} from "@/lib/candidate-types";
import { validateCandidateForm } from "@/lib/candidate-validators";
import type { Job } from "@/lib/job-types";
import { listJobs } from "@/services/job.service";
import { downloadResumeFile, viewResumeFile } from "@/lib/resume-file";

interface CandidateFormProps {
  initialValues: CandidateFormValues;
  submitLabel: string;
  loading?: boolean;
  onSubmit: (values: CandidateFormValues) => Promise<void>;
  onCancel?: () => void;
  /** Display-only uploaded resume filename (never show IDs/paths). */
  resumeFileName?: string;
  /** Authenticated API path for viewing/downloading the resume file. */
  resumeViewUrl?: string;
}

function SelectField({
  label,
  name,
  value,
  options,
  error,
  onChange,
}: {
  label: string;
  name: string;
  value: string;
  options: { value: string; label: string }[];
  error?: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={name} className="block text-sm font-medium text-[var(--text-secondary)]">
        {label}
      </label>
      <select
        id={name}
        name={name}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={error ? true : undefined}
        className={`ats-input ${error ? "!border-red-400 focus:!ring-red-200" : ""}`}
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {error && <p className="text-xs font-medium text-red-600">{error}</p>}
    </div>
  );
}

export function CandidateForm({
  initialValues,
  submitLabel,
  loading,
  onSubmit,
  onCancel,
  resumeFileName,
  resumeViewUrl,
}: CandidateFormProps) {
  const [values, setValues] = useState(initialValues);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [jobs, setJobs] = useState<Job[]>([]);
  const [openingResume, setOpeningResume] = useState(false);
  const [downloadingResume, setDownloadingResume] = useState(false);

  useEffect(() => {
    setValues(initialValues);
  }, [initialValues]);

  useEffect(() => {
    listJobs({ page: 1, limit: 100 })
      .then((result) => setJobs(result.jobs))
      .catch(() => setJobs([]));
  }, []);

  function update(field: keyof CandidateFormValues, value: string) {
    setValues((prev) => ({ ...prev, [field]: value }));
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const fieldErrors = validateCandidateForm(values);
    setErrors(fieldErrors);
    if (Object.keys(fieldErrors).length) return;
    await onSubmit(values);
  }

  async function handleViewResume() {
    if (!resumeViewUrl) return;
    setOpeningResume(true);
    try {
      await viewResumeFile(resumeViewUrl);
    } catch {
      setErrors((prev) => ({ ...prev, resume: "Unable to open resume file" }));
    } finally {
      setOpeningResume(false);
    }
  }

  async function handleDownloadResume() {
    if (!resumeViewUrl) return;
    setDownloadingResume(true);
    try {
      await downloadResumeFile(resumeViewUrl, resumeFileName);
    } catch {
      setErrors((prev) => ({ ...prev, resume: "Unable to download resume file" }));
    } finally {
      setDownloadingResume(false);
    }
  }

  const showUploadedResume = Boolean(resumeFileName && resumeViewUrl);

  return (
    <form onSubmit={handleSubmit} className="space-y-5" noValidate>
      <SelectField
        label="Job (optional)"
        name="job_id"
        value={values.job_id}
        options={[
          { value: "", label: "No job linked" },
          ...jobs.map((job) => ({ value: job.id, label: job.title })),
        ]}
        onChange={(v) => update("job_id", v)}
      />

      <div className="grid gap-5 sm:grid-cols-2">
        <Input
          label="Name"
          name="name"
          value={values.name}
          onChange={(e) => update("name", e.target.value)}
          error={errors.name}
          placeholder="Jane Doe"
        />
        <Input
          label="Email"
          name="email"
          type="email"
          value={values.email}
          onChange={(e) => update("email", e.target.value)}
          error={errors.email}
          placeholder="jane@example.com"
        />
      </div>

      <div className="grid gap-5 sm:grid-cols-2">
        <Input
          label="Phone"
          name="phone"
          value={values.phone}
          onChange={(e) => update("phone", e.target.value)}
          placeholder="+1 555 0100"
        />
        <Input
          label="Experience (years)"
          name="experience_years"
          type="number"
          min={0}
          value={values.experience_years}
          onChange={(e) => update("experience_years", e.target.value)}
          error={errors.experience_years}
          placeholder="5"
        />
      </div>

      <div className="grid gap-5 sm:grid-cols-2">
        <Input
          label="Current Company"
          name="current_company"
          value={values.current_company}
          onChange={(e) => update("current_company", e.target.value)}
        />
        <Input
          label="Current Job Title"
          name="current_designation"
          value={values.current_designation}
          onChange={(e) => update("current_designation", e.target.value)}
        />
      </div>

      <Input
        label="Location"
        name="location"
        value={values.location}
        onChange={(e) => update("location", e.target.value)}
        placeholder="City, Country"
      />

      <Input
        label="Skills"
        name="skills"
        value={values.skills}
        onChange={(e) => update("skills", e.target.value)}
        placeholder="Go, PostgreSQL, React"
      />
      <p className="-mt-3 text-xs text-[var(--text-muted)]">Separate skills with commas</p>

      <div className="space-y-1.5">
        <label htmlFor="education" className="block text-sm font-medium text-[var(--text-secondary)]">
          Education
        </label>
        <textarea
          id="education"
          rows={3}
          value={values.education}
          onChange={(e) => update("education", e.target.value)}
          className="ats-input"
          placeholder={"Degree: B.S. | Branch: Computer Science | School: Example University | Years: 2015 - 2019"}
        />
      </div>

      <div className="space-y-1.5">
        <label htmlFor="certifications" className="block text-sm font-medium text-[var(--text-secondary)]">
          Certifications
        </label>
        <textarea
          id="certifications"
          rows={3}
          value={values.certifications}
          onChange={(e) => update("certifications", e.target.value)}
          className="ats-input"
          placeholder={"AWS Certified Solutions Architect\nOne certification per line"}
        />
      </div>

      <SelectField
        label="Status"
        name="status"
        value={values.status}
        options={CANDIDATE_STATUSES}
        onChange={(v) => update("status", v)}
      />

      <Input
        label="Source"
        name="source"
        value={values.source}
        onChange={(e) => update("source", e.target.value)}
        placeholder="LinkedIn, Referral, Website"
      />

      <div className="rounded-2xl bg-[var(--surface-muted)] p-4 ring-1 ring-[var(--border)] sm:p-5">
        <p className="text-sm font-semibold text-[var(--text-primary)]">Resume</p>
        <div className="mt-4 space-y-4">
          {showUploadedResume ? (
            <div className="space-y-3 rounded-xl bg-[var(--surface)] px-4 py-3 ring-1 ring-[var(--border)]">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="truncate text-sm font-medium text-[var(--text-primary)]">{resumeFileName}</p>
                <div className="flex flex-wrap gap-2">
                  <Button
                    type="button"
                    variant="secondary"
                    className="w-auto px-4"
                    loading={openingResume}
                    onClick={handleViewResume}
                  >
                    View Resume
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    className="w-auto px-4"
                    loading={downloadingResume}
                    onClick={handleDownloadResume}
                  >
                    Download Resume
                  </Button>
                </div>
              </div>
              <p className="text-xs font-medium text-emerald-700">Upload successful</p>
            </div>
          ) : (
            <>
              <Input
                label="Resume URL"
                name="resume_url"
                value={values.resume_url}
                onChange={(e) => update("resume_url", e.target.value)}
                placeholder="https://..."
              />
              <div className="space-y-1.5">
                <label htmlFor="resume_text" className="block text-sm font-medium text-[var(--text-secondary)]">
                  Resume Text
                </label>
                <textarea
                  id="resume_text"
                  rows={3}
                  value={values.resume_text}
                  onChange={(e) => update("resume_text", e.target.value)}
                  className="ats-input"
                  placeholder="Plain text resume content..."
                />
              </div>
            </>
          )}

          <div className="space-y-1.5">
            <label htmlFor="resume_summary" className="block text-sm font-medium text-[var(--text-secondary)]">
              Resume Summary
            </label>
            <textarea
              id="resume_summary"
              rows={4}
              value={values.resume_summary}
              onChange={(e) => update("resume_summary", e.target.value)}
              className="ats-input whitespace-pre-wrap"
              placeholder="Short summary..."
            />
          </div>

          {!showUploadedResume && (
            <div className="grid gap-4 sm:grid-cols-2">
              <SelectField
                label="Parsing Status"
                name="parsing_status"
                value={values.parsing_status}
                options={PROCESSING_STATUSES}
                onChange={(v) => update("parsing_status", v)}
              />
              <SelectField
                label="Embedding Status"
                name="embedding_status"
                value={values.embedding_status}
                options={PROCESSING_STATUSES}
                onChange={(v) => update("embedding_status", v)}
              />
            </div>
          )}
          {errors.resume && <p className="text-xs font-medium text-red-600">{errors.resume}</p>}
        </div>
      </div>

      <div className="ats-form-footer !mt-6 !border-t-0 !px-0 sm:!justify-between">
        {onCancel ? (
          <Button type="button" variant="ghost" className="order-2 w-auto sm:order-1" onClick={onCancel}>
            Cancel
          </Button>
        ) : (
          <span />
        )}
        <div className="order-1 flex w-full flex-col gap-2 sm:order-2 sm:w-auto sm:flex-row">
          <Button type="submit" loading={loading} className="sm:min-w-[9rem]">
            {submitLabel}
          </Button>
        </div>
      </div>
    </form>
  );
}
