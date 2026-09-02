import { EditCandidateView } from "@/components/candidates/EditCandidateView";

export default async function EditCandidatePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <EditCandidateView candidateId={id} />;
}
