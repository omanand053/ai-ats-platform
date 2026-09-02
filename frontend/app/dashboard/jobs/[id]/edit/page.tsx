import { EditJobView } from "@/components/jobs/EditJobView";

export default async function EditJobPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <EditJobView jobId={id} />;
}
