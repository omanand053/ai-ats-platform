"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useState } from "react";
import { CandidateForm } from "@/components/candidates/CandidateForm";
import { ResumeUploadScreen } from "@/components/candidates/ResumeUploadScreen";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Spinner } from "@/components/ui/Spinner";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import { emptyCandidateForm, type CandidateFormValues } from "@/lib/candidate-types";
import {
  formatCertificationsForInput,
  formatEducationForInput,
  formatProjectsForInput,
  formatSkillsForInput,
  formValuesToPayload,
} from "@/lib/candidate-validators";
import type { ResumeUploadResult } from "@/lib/resume-types";
import { createCandidate } from "@/services/candidate.service";
import { attachResume } from "@/services/resume.service";

function parsedToFormValues(result: ResumeUploadResult): CandidateFormValues {
  const p = result.parsed;
  return {
    ...emptyCandidateForm,
    name: p.full_name || "",
    email: p.email || "",
    phone: p.phone || "",
    experience_years:
      p.experience_years !== undefined && p.experience_years !== null
        ? String(p.experience_years)
        : "",
    current_company: p.current_company || "",
    current_designation: p.current_designation || "",
    location: p.location || "",
    skills: formatSkillsForInput(p.skills || []),
    education: formatEducationForInput(p.education || []),
    certifications: formatCertificationsForInput(p.certifications || []),
    projects: formatProjectsForInput(p.projects || []),
    resume_url: result.file_url,
    resume_text: p.raw_text || "",
    resume_summary: p.summary || "",
    source: "resume_upload",
    status: "applied",
    parsing_status: result.parsing_status === "completed" ? "completed" : "failed",
    embedding_status: "pending",
  };
}

export function CreateCandidateView() {
  const ready = useRequireAuth();
  const router = useRouter();
  const { toasts, show, dismiss } = useToast();
  const [step, setStep] = useState<"upload" | "review">("upload");
  const [formValues, setFormValues] = useState<CandidateFormValues>(emptyCandidateForm);
  const [resumeMeta, setResumeMeta] = useState<{
    resumeId: string;
    fileName: string;
    fileUrl: string;
  } | null>(null);
  const [loading, setLoading] = useState(false);

  const handleUploadComplete = useCallback(
    (result: ResumeUploadResult) => {
      setResumeMeta({
        resumeId: result.resume_id,
        fileName: result.file_name,
        fileUrl: result.file_url,
      });
      setFormValues(parsedToFormValues(result));
      if (result.parsing_status === "failed") {
        show("Resume uploaded, but parsing could not extract text. Please fill in details manually.", "error");
      } else {
        show("Resume parsed successfully. Review and save the candidate.");
      }
      setStep("review");
    },
    [show],
  );

  async function handleSubmit(values: CandidateFormValues) {
    setLoading(true);
    try {
      const payload = formValuesToPayload(values);
      if (resumeMeta?.fileUrl) {
        payload.resume_url = resumeMeta.fileUrl;
      }
      if (!payload.source) {
        payload.source = "resume_upload";
      }
      if (resumeMeta) {
        payload.parsing_status =
          values.parsing_status === "completed" ? "completed" : values.parsing_status;
      }

      const candidate = await createCandidate(payload);
      let attachFailed = false;
      if (resumeMeta?.resumeId) {
        try {
          await attachResume(resumeMeta.resumeId, candidate.id);
        } catch {
          attachFailed = true;
          show("Candidate saved, but resume could not be linked. You can still view it via the resume URL.", "error");
        }
      }
      if (!attachFailed) {
        show("Candidate created successfully!");
      }
      router.push(`/dashboard/candidates/${candidate.id}`);
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to create candidate";
      show(message, "error");
    } finally {
      setLoading(false);
    }
  }

  if (!ready) {
    return <Spinner />;
  }

  return (
    <>
      <>
        <Link href="/dashboard/candidates" className="ats-link mb-2 inline-block text-sm">
          ← Back to candidates
        </Link>
        <PageHeader
          title="Add Candidate"
          subtitle={
            step === "upload"
              ? "Upload a resume to create a candidate profile"
              : "Review parsed details, edit if needed, then save"
          }
        />

        {step === "upload" ? (
          <Card padding="lg">
            <ResumeUploadScreen
              onUploadComplete={handleUploadComplete}
              onCancel={() => router.push("/dashboard/candidates")}
            />
          </Card>
        ) : (
          <Card padding="lg">
            <CandidateForm
              initialValues={formValues}
              submitLabel="Save Candidate"
              loading={loading}
              onSubmit={handleSubmit}
              onCancel={() => router.push("/dashboard/candidates")}
              resumeFileName={resumeMeta?.fileName}
              resumeViewUrl={resumeMeta?.fileUrl}
            />
          </Card>
        )}
      </>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
