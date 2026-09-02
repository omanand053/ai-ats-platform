"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { JobForm } from "@/components/jobs/JobForm";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Spinner } from "@/components/ui/Spinner";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import { emptyJobForm, formValuesToPayload } from "@/lib/job-validators";
import type { JobFormValues } from "@/lib/job-types";
import { createJob } from "@/services/job.service";
import { useState } from "react";

export function CreateJobView() {
  const ready = useRequireAuth();
  const router = useRouter();
  const { toasts, show, dismiss } = useToast();
  const [loading, setLoading] = useState(false);

  async function handleSubmit(values: JobFormValues) {
    setLoading(true);
    try {
      const job = await createJob(formValuesToPayload(values));
      show("Job created successfully!");
      router.push(`/dashboard/jobs/${job.id}`);
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to create job";
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
        <Link href="/dashboard/jobs" className="ats-link mb-2 inline-block text-sm">
          ← Back to jobs
        </Link>
        <PageHeader
          title="Create Job"
          subtitle="Add a new job posting to your company"
        />

        <Card padding="lg">
          <JobForm
            initialValues={emptyJobForm}
            submitLabel="Create Job"
            loading={loading}
            onSubmit={handleSubmit}
            onCancel={() => router.push("/dashboard/jobs")}
          />
        </Card>
      </>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
