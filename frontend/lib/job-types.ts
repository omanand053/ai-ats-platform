export type JobStatus = "draft" | "open" | "closed";

export type EmploymentType =
  | "full_time"
  | "part_time"
  | "contract"
  | "internship"
  | "temporary";

export interface Job {
  id: string;
  company_id: string;
  title: string;
  department?: string;
  location?: string;
  employment_type: EmploymentType;
  experience_required?: string;
  description?: string;
  required_skills: string[];
  status: JobStatus;
  embedding_status?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface JobListResult {
  jobs: Job[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface JobPayload {
  title: string;
  department?: string;
  location?: string;
  employment_type: EmploymentType;
  experience_required?: string;
  description?: string;
  required_skills: string[];
  status: JobStatus;
}

export interface JobFormValues {
  title: string;
  department: string;
  location: string;
  employment_type: EmploymentType;
  experience_required: string;
  description: string;
  required_skills: string;
  status: JobStatus;
}

export const EMPLOYMENT_TYPES: { value: EmploymentType; label: string }[] = [
  { value: "full_time", label: "Full Time" },
  { value: "part_time", label: "Part Time" },
  { value: "contract", label: "Contract" },
  { value: "internship", label: "Internship" },
  { value: "temporary", label: "Temporary" },
];

export const JOB_STATUSES: { value: JobStatus; label: string }[] = [
  { value: "draft", label: "Draft" },
  { value: "open", label: "Open" },
  { value: "closed", label: "Closed" },
];
