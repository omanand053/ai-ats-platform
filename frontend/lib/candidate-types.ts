export type CandidateStatus =
  | "applied"
  | "screening"
  | "shortlisted"
  | "recruiter_shortlisted"
  | "ai_shortlisted"
  | "interview"
  | "selected"
  | "offer"
  | "hired"
  | "rejected";

export type ProcessingStatus = "pending" | "processing" | "completed" | "failed";

export interface FitScoreBreakdown {
  skills: number;
  experience: number;
  education: number;
  seniority: number;
  location: number;
  certifications: number;
}

export interface Candidate {
  id: string;
  company_id: string;
  job_id?: string;
  name: string;
  email: string;
  phone?: string;
  experience_years?: number;
  current_company?: string;
  current_designation?: string;
  location?: string;
  skills: string[];
  status: CandidateStatus;
  resume_url?: string;
  resume_text?: string;
  resume_summary?: string;
  source?: string;
  parsing_status: ProcessingStatus;
  embedding_status: ProcessingStatus;
  overall_score?: number;
  score_breakdown?: FitScoreBreakdown;
  matched_skills?: string[];
  missing_skills?: string[];
  last_scored_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CandidateListResult {
  candidates: Candidate[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface CandidatePayload {
  job_id?: string;
  name: string;
  email: string;
  phone?: string;
  experience_years?: number;
  current_company?: string;
  current_designation?: string;
  location?: string;
  skills: string[];
  status: CandidateStatus;
  resume_url?: string;
  resume_text?: string;
  resume_summary?: string;
  source?: string;
  parsing_status?: ProcessingStatus;
  embedding_status?: ProcessingStatus;
}

export interface CandidateFormValues {
  job_id: string;
  name: string;
  email: string;
  phone: string;
  experience_years: string;
  current_company: string;
  current_designation: string;
  location: string;
  skills: string;
  status: CandidateStatus;
  education: string;
  certifications: string;
  /** Editable projects block: Name | tech1, tech2\nDescription\n---\n... */
  projects: string;
  resume_url: string;
  resume_text: string;
  resume_summary: string;
  source: string;
  parsing_status: ProcessingStatus;
  embedding_status: ProcessingStatus;
}

export const CANDIDATE_STATUSES: { value: CandidateStatus; label: string }[] = [
  { value: "applied", label: "Applied" },
  { value: "screening", label: "Screening" },
  { value: "shortlisted", label: "Shortlisted" },
  { value: "ai_shortlisted", label: "AI Shortlisted" },
  { value: "recruiter_shortlisted", label: "Recruiter Shortlisted" },
  { value: "interview", label: "Interview" },
  { value: "selected", label: "Selected" },
  { value: "offer", label: "Offer" },
  { value: "hired", label: "Hired" },
  { value: "rejected", label: "Rejected" },
];

export const PROCESSING_STATUSES: { value: ProcessingStatus; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "processing", label: "Processing" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
];

export const emptyCandidateForm: CandidateFormValues = {
  job_id: "",
  name: "",
  email: "",
  phone: "",
  experience_years: "",
  current_company: "",
  current_designation: "",
  location: "",
  skills: "",
  status: "applied",
  education: "",
  certifications: "",
  projects: "",
  resume_url: "",
  resume_text: "",
  resume_summary: "",
  source: "",
  parsing_status: "pending",
  embedding_status: "pending",
};
