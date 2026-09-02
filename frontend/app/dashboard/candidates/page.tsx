import { Suspense } from "react";
import { CandidatesListView } from "@/components/candidates/CandidatesListView";
import { TableSkeleton } from "@/components/ui/Skeleton";

export default function CandidatesPage() {
  return (
    <Suspense fallback={<TableSkeleton rows={6} />}>
      <CandidatesListView />
    </Suspense>
  );
}
