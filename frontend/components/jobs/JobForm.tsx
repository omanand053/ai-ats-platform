"use client";

import { FormEvent, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import {
  EMPLOYMENT_TYPES,
  JOB_STATUSES,
  type JobFormValues,
} from "@/lib/job-types";
import { validateJobForm } from "@/lib/job-validators";

interface JobFormProps {
  initialValues: JobFormValues;
  submitLabel: string;
  loading?: boolean;
  onSubmit: (values: JobFormValues) => Promise<void>;
  onCancel?: () => void;
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
      <label htmlFor={name} className="block text-sm font-medium text-slate-700">
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

export function JobForm({
  initialValues,
  submitLabel,
  loading,
  onSubmit,
  onCancel,
}: JobFormProps) {
  const [values, setValues] = useState(initialValues);
  const [errors, setErrors] = useState<Record<string, string>>({});

  function update(field: keyof JobFormValues, value: string) {
    setValues((prev) => ({ ...prev, [field]: value }));
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const fieldErrors = validateJobForm(values);
    setErrors(fieldErrors);
    if (Object.keys(fieldErrors).length) return;
    await onSubmit(values);
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5" noValidate>
      <Input
        label="Title"
        name="title"
        value={values.title}
        onChange={(e) => update("title", e.target.value)}
        error={errors.title}
        placeholder="Senior Go Engineer"
      />

      <div className="grid gap-5 sm:grid-cols-2">
        <Input
          label="Department"
          name="department"
          value={values.department}
          onChange={(e) => update("department", e.target.value)}
          placeholder="Engineering"
        />
        <Input
          label="Location"
          name="location"
          value={values.location}
          onChange={(e) => update("location", e.target.value)}
          placeholder="Remote"
        />
      </div>

      <div className="grid gap-5 sm:grid-cols-2">
        <SelectField
          label="Employment Type"
          name="employment_type"
          value={values.employment_type}
          options={EMPLOYMENT_TYPES}
          error={errors.employment_type}
          onChange={(v) => update("employment_type", v)}
        />
        <SelectField
          label="Status"
          name="status"
          value={values.status}
          options={JOB_STATUSES}
          error={errors.status}
          onChange={(v) => update("status", v)}
        />
      </div>

      <Input
        label="Experience Required"
        name="experience_required"
        value={values.experience_required}
        onChange={(e) => update("experience_required", e.target.value)}
        placeholder="5+ years"
      />

      <div className="space-y-1.5">
        <label htmlFor="description" className="block text-sm font-medium text-[var(--text-secondary)]">
          Description
        </label>
        <textarea
          id="description"
          name="description"
          rows={4}
          value={values.description}
          onChange={(e) => update("description", e.target.value)}
          placeholder="Describe the role and responsibilities..."
          className="ats-input"
        />
      </div>

      <Input
        label="Required Skills"
        name="required_skills"
        value={values.required_skills}
        onChange={(e) => update("required_skills", e.target.value)}
        placeholder="Go, PostgreSQL, Docker"
      />
      <p className="-mt-3 text-xs text-[var(--text-muted)]">Separate skills with commas</p>

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
