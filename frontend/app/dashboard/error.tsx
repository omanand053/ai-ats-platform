"use client";

import { useEffect } from "react";
import { ErrorState } from "@/components/ui/ErrorState";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    if (process.env.NODE_ENV === "development") {
      console.error(error);
    }
  }, [error]);

  return (
    <>
      <ErrorState
        title="Something interrupted this workspace"
        description={
          error.message && !/something went wrong/i.test(error.message)
            ? error.message
            : "We hit an unexpected problem loading this view. Retry the action or refresh to continue."
        }
        onRetry={reset}
      />
    </>
  );
}
