"use client";

import { useEffect } from "react";
import { ErrorState } from "@/components/ui/ErrorState";

export default function GlobalError({
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
    <html lang="en">
      <body className="min-h-full bg-[var(--background,#f7f8fa)] font-sans text-[var(--foreground,#0b1220)]">
        <ErrorState
          title="Application error"
          description={
            error.message && !/something went wrong/i.test(error.message)
              ? error.message
              : "The application ran into a problem. Retry or refresh to recover."
          }
          onRetry={reset}
          homeHref="/"
        />
      </body>
    </html>
  );
}
