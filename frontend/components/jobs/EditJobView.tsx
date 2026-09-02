"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { JobForm } from "@/components/jobs/JobForm";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Spinner } from "@/components/ui/Spinner";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import { formValuesToPayload, jobToFormValues } from "@/lib/job-validators";
import type { JobFormValues } from "@/lib/job-types";
import { getJob, updateJob } from "@/services/job.service";

export function EditJobView({ jobId }: { jobId: string }) {
  const ready = useRequireAuth();
  const router = useRouter();
  const { toasts, show, dismiss } = useToast();
  const [initialValues, setInitialValues] = useState<JobFormValues | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!ready) return;
    getJob(jobId)
      .then((job) => setInitialValues(jobToFormValues(job)))
      .catch((err) => {
        const message = err instanceof ApiClientError ? err.message : "Failed to load job";
        show(message, "error");
      })
      .finally(() => setLoading(false));
  }, [ready, jobId, show]);

  async function handleSubmit(values: JobFormValues) {
    setSaving(true);
    try {
      await updateJob(jobId, formValuesToPayload(values));
      show("Job updated successfully!");
      router.push(`/dashboard/jobs/${jobId}`);
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to update job";
      show(message, "error");
    } finally {
      setSaving(false);
    }
  }

  if (!ready || loading) {
    return <Spinner label="Loading job..." />;
  }

  if (!initialValues) {
    return (
      <>
        <>
          <Card className="text-center">
            <p className="text-sm text-red-600">Job not found</p>
          </Card>
        </>
        <ToastContainer toasts={toasts} onDismiss={dismiss} />
      </>
    );
  }

  return (
    <>
      <>
        <Link href={`/dashboard/jobs/${jobId}`} className="ats-link mb-2 inline-block text-sm">
          ← Back to job
        </Link>
        <PageHeader title="Edit Job" subtitle="Update job posting details" />

        <Card padding="lg">
          <JobForm
            key={jobId}
            initialValues={initialValues}
            submitLabel="Save Changes"
            loading={saving}
            onSubmit={handleSubmit}
            onCancel={() => router.push(`/dashboard/jobs/${jobId}`)}
          />
        </Card>
      </>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
