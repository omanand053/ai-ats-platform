import { CandidateDetailView } from "@/components/candidates/CandidateDetailView";

export default async function CandidateDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <CandidateDetailView candidateId={id} />;
}
