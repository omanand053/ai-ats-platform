export type ProcessingStatus = "pending" | "processing" | "completed" | "failed";

export interface ParsedProject {
  name: string;
  technologies: string[];
  description: string;
}

export interface ParsedEducation {
  school: string;
  degree: string;
  branch: string;
  years: string;
}

export interface ParsedResume {
  full_name: string;
  email: string;
  phone: string;
  experience_years?: number;
  skills: string[];
  projects: ParsedProject[];
  education: ParsedEducation[];
  certifications: string[];
  current_company: string;
  current_designation: string;
  location?: string;
  summary: string;
  raw_text?: string;
}

export interface ResumeUploadResult {
  resume_id: string;
  file_name: string;
  file_url: string;
  parsing_status: ProcessingStatus;
  parsed: ParsedResume;
}
