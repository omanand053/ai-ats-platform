"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { CandidateForm } from "@/components/candidates/CandidateForm";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Spinner } from "@/components/ui/Spinner";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import { candidateToFormValues, formValuesToPayload } from "@/lib/candidate-validators";
import type { CandidateFormValues } from "@/lib/candidate-types";
import { getCandidate, updateCandidate } from "@/services/candidate.service";

export function EditCandidateView({ candidateId }: { candidateId: string }) {
  const ready = useRequireAuth();
  const router = useRouter();
  const { toasts, show, dismiss } = useToast();
  const [initialValues, setInitialValues] = useState<CandidateFormValues | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!ready) return;
    getCandidate(candidateId)
      .then((c) => setInitialValues(candidateToFormValues(c)))
      .catch((err) => {
        const message = err instanceof ApiClientError ? err.message : "Failed to load candidate";
        show(message, "error");
      })
      .finally(() => setLoading(false));
  }, [ready, candidateId, show]);

  async function handleSubmit(values: CandidateFormValues) {
    setSaving(true);
    try {
      await updateCandidate(candidateId, formValuesToPayload(values));
      show("Candidate updated successfully!");
      router.push(`/dashboard/candidates/${candidateId}`);
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to update candidate";
      show(message, "error");
    } finally {
      setSaving(false);
    }
  }

  if (!ready || loading) {
    return <Spinner label="Loading candidate..." />;
  }

  if (!initialValues) {
    return (
      <>
        <>
          <Card className="text-center">
            <p className="text-sm text-red-600">Candidate not found</p>
          </Card>
        </>
        <ToastContainer toasts={toasts} onDismiss={dismiss} />
      </>
    );
  }

  return (
    <>
      <>
        <Link
          href={`/dashboard/candidates/${candidateId}`}
          className="ats-link mb-2 inline-block text-sm"
        >
          ← Back to candidate
        </Link>
        <PageHeader title="Edit Candidate" subtitle="Update candidate details" />

        <Card padding="lg">
          <CandidateForm
            key={candidateId}
            initialValues={initialValues}
            submitLabel="Save Changes"
            loading={saving}
            onSubmit={handleSubmit}
            onCancel={() => router.push(`/dashboard/candidates/${candidateId}`)}
          />
        </Card>
      </>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
