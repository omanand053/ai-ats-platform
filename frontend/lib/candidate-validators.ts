import type { FieldErrors } from "@/lib/validators";
import type {
  Candidate,
  CandidateFormValues,
  CandidatePayload,
} from "@/lib/candidate-types";
import type { ParsedEducation, ParsedProject } from "@/lib/resume-types";
import { dedupeSkills } from "@/lib/skill-normalize";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const META_START = "---ATS_META_V1---";
const META_END = "---ATS_RAW---";

type ResumeMeta = {
  education: ParsedEducation[];
  certifications: string[];
  projects: ParsedProject[];
};

export function validateCandidateForm(values: CandidateFormValues): FieldErrors {
  const errors: FieldErrors = {};
  if (!values.name.trim()) errors.name = "Name is required";
  if (!values.email.trim()) errors.email = "Email is required";
  else if (!EMAIL_RE.test(values.email)) errors.email = "Enter a valid email";
  if (values.experience_years && Number.isNaN(Number(values.experience_years))) {
    errors.experience_years = "Enter a valid number";
  }
  return errors;
}

export function parseSkillsInput(input: string): string[] {
  return dedupeSkills(
    input
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean),
  );
}

export function formatSkillsForInput(skills: string[]): string {
  return dedupeSkills(skills).join(", ");
}

export function formatProjectsForInput(projects: ParsedProject[]): string {
  return projects
    .map((p) => {
      const techs = (p.technologies || []).join(", ");
      const header = techs ? `${p.name} | ${techs}` : p.name;
      const desc = (p.description || "").trim();
      return desc ? `${header}\n${desc}` : header;
    })
    .join("\n---\n");
}

export function parseProjectsInput(input: string): ParsedProject[] {
  if (!input.trim()) return [];
  return input
    .split(/\n---\n/)
    .map((chunk) => chunk.trim())
    .filter(Boolean)
    .map((chunk) => {
      const lines = chunk.split("\n").map((l) => l.trim()).filter(Boolean);
      const header = lines[0] || "Untitled project";
      const [namePart, techPart] = header.split("|").map((s) => s.trim());
      const technologies = techPart
        ? techPart.split(",").map((t) => t.trim()).filter(Boolean)
        : [];
      return {
        name: namePart || "Untitled project",
        technologies,
        description: lines.slice(1).join("\n"),
      };
    });
}

export function formatEducationForInput(education: ParsedEducation[]): string {
  return education
    .map((e) => {
      const bits = [
        e.degree && `Degree: ${e.degree}`,
        e.branch && `Branch: ${e.branch}`,
        e.school && `School: ${e.school}`,
        e.years && `Years: ${e.years}`,
      ].filter(Boolean);
      return bits.join(" | ");
    })
    .filter(Boolean)
    .join("\n");
}

export function parseEducationInput(input: string): ParsedEducation[] {
  return input
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)
    .map((line) => parseEducationLine(line))
    .filter((e) => e.school || e.degree || e.branch || e.years);
}

export function formatCertificationsForInput(certs: string[]): string {
  return (certs || []).map((c) => c.trim()).filter(Boolean).join("\n");
}

export function parseCertificationsInput(input: string): string[] {
  return input
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
}

function parseEducationLine(line: string): ParsedEducation {
  const empty: ParsedEducation = { school: "", degree: "", branch: "", years: "" };
  if (line.includes(":")) {
    const entry = { ...empty };
    for (const chunk of line.split(/[|;]/)) {
      const idx = chunk.indexOf(":");
      if (idx < 0) continue;
      const key = chunk.slice(0, idx).trim().toLowerCase();
      const val = chunk.slice(idx + 1).trim();
      if (!val) continue;
      if (/school|college|university|institute/.test(key)) entry.school = val;
      else if (/degree|qualification/.test(key)) entry.degree = val;
      else if (/branch|major|specialization|stream/.test(key)) entry.branch = val;
      else if (/year|duration/.test(key)) entry.years = val;
    }
    if (entry.school || entry.degree || entry.branch || entry.years) return entry;
  }

  if (typeof line === "string" && line.trim()) {
    return { ...empty, school: line.trim() };
  }
  return empty;
}

function normalizeEducationList(raw: unknown): ParsedEducation[] {
  if (!Array.isArray(raw)) return [];
  return raw
    .map((item) => {
      if (typeof item === "string") {
        return parseEducationLine(item);
      }
      if (item && typeof item === "object") {
        const o = item as Record<string, unknown>;
        return {
          school: String(o.school ?? ""),
          degree: String(o.degree ?? ""),
          branch: String(o.branch ?? ""),
          years: String(o.years ?? ""),
        };
      }
      return { school: "", degree: "", branch: "", years: "" };
    })
    .filter((e) => e.school || e.degree || e.branch || e.years);
}

function encodeResumeText(
  rawText: string,
  education: ParsedEducation[],
  certifications: string[],
  projects: ParsedProject[],
): string {
  const meta: ResumeMeta = { education, certifications, projects };
  const raw = rawText.trim();
  return `${META_START}\n${JSON.stringify(meta)}\n${META_END}\n${raw}`;
}

export function decodeResumeExtras(resumeText?: string): {
  education: ParsedEducation[];
  certifications: string[];
  projects: ParsedProject[];
  rawText: string;
} {
  const text = resumeText ?? "";
  if (!text.includes(META_START) || !text.includes(META_END)) {
    return { education: [], certifications: [], projects: [], rawText: text };
  }
  try {
    const start = text.indexOf(META_START) + META_START.length;
    const end = text.indexOf(META_END);
    const metaRaw = text.slice(start, end).trim();
    const rawText = text.slice(end + META_END.length).replace(/^\n/, "");
    const meta = JSON.parse(metaRaw) as ResumeMeta;
    return {
      education: normalizeEducationList(meta.education),
      certifications: Array.isArray(meta.certifications)
        ? meta.certifications.map(String).filter(Boolean)
        : [],
      projects: Array.isArray(meta.projects) ? meta.projects : [],
      rawText,
    };
  } catch {
    return { education: [], certifications: [], projects: [], rawText: text };
  }
}

export function formValuesToPayload(values: CandidateFormValues): CandidatePayload {
  const education = parseEducationInput(values.education);
  const certifications = parseCertificationsInput(values.certifications);
  const rawText = values.resume_text.trim();

  const payload: CandidatePayload = {
    name: values.name.trim(),
    email: values.email.trim(),
    skills: parseSkillsInput(values.skills),
    status: values.status,
    parsing_status: values.parsing_status,
    embedding_status: values.embedding_status,
  };

  if (values.job_id) payload.job_id = values.job_id;
  if (values.phone.trim()) payload.phone = values.phone.trim();
  if (values.experience_years.trim()) {
    payload.experience_years = Number(values.experience_years);
  }
  if (values.current_company.trim()) payload.current_company = values.current_company.trim();
  if (values.current_designation.trim()) {
    payload.current_designation = values.current_designation.trim();
  }
  if (values.location.trim()) payload.location = values.location.trim();
  if (values.resume_url.trim()) payload.resume_url = values.resume_url.trim();
  if (values.resume_summary.trim()) payload.resume_summary = values.resume_summary.trim();
  if (values.source.trim()) payload.source = values.source.trim();

  // Persist education/certifications/projects inside resume_text without Candidate CRUD API changes.
  const projects = parseProjectsInput(values.projects);
  if (rawText || education.length || certifications.length || projects.length) {
    payload.resume_text = encodeResumeText(rawText, education, certifications, projects);
  }

  return payload;
}

export function candidateToFormValues(candidate: Candidate): CandidateFormValues {
  const extras = decodeResumeExtras(candidate.resume_text);
  return {
    job_id: candidate.job_id ?? "",
    name: candidate.name,
    email: candidate.email,
    phone: candidate.phone ?? "",
    experience_years:
      candidate.experience_years !== undefined ? String(candidate.experience_years) : "",
    current_company: candidate.current_company ?? "",
    current_designation: candidate.current_designation ?? "",
    location: candidate.location ?? "",
    skills: formatSkillsForInput(candidate.skills ?? []),
    status: candidate.status,
    education: formatEducationForInput(extras.education),
    certifications: formatCertificationsForInput(extras.certifications),
    projects: formatProjectsForInput(extras.projects),
    resume_url: candidate.resume_url ?? "",
    resume_text: extras.rawText,
    resume_summary: candidate.resume_summary ?? "",
    source: candidate.source ?? "",
    parsing_status: candidate.parsing_status,
    embedding_status: candidate.embedding_status,
  };
}
