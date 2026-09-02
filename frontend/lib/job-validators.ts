import type { FieldErrors } from "@/lib/validators";
import type { JobFormValues } from "@/lib/job-types";

export function validateJobForm(values: JobFormValues): FieldErrors {
  const errors: FieldErrors = {};
  if (!values.title.trim()) errors.title = "Title is required";
  if (!values.employment_type) errors.employment_type = "Employment type is required";
  if (!values.status) errors.status = "Status is required";
  return errors;
}

export function parseSkillsInput(input: string): string[] {
  return input
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

export function formatSkillsForInput(skills: string[]): string {
  return skills.join(", ");
}

export function formValuesToPayload(values: JobFormValues) {
  return {
    title: values.title.trim(),
    department: values.department.trim() || undefined,
    location: values.location.trim() || undefined,
    employment_type: values.employment_type,
    experience_required: values.experience_required.trim() || undefined,
    description: values.description.trim() || undefined,
    required_skills: parseSkillsInput(values.required_skills),
    status: values.status,
  };
}

export function jobToFormValues(job: {
  title: string;
  department?: string;
  location?: string;
  employment_type: JobFormValues["employment_type"];
  experience_required?: string;
  description?: string;
  required_skills: string[];
  status: JobFormValues["status"];
}): JobFormValues {
  return {
    title: job.title,
    department: job.department ?? "",
    location: job.location ?? "",
    employment_type: job.employment_type,
    experience_required: job.experience_required ?? "",
    description: job.description ?? "",
    required_skills: formatSkillsForInput(job.required_skills ?? []),
    status: job.status,
  };
}

export const emptyJobForm: JobFormValues = {
  title: "",
  department: "",
  location: "",
  employment_type: "full_time",
  experience_required: "",
  description: "",
  required_skills: "",
  status: "draft",
};
