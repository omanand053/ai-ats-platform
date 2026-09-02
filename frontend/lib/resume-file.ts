import { getToken } from "@/lib/auth-storage";

async function fetchResumeBlob(url: string): Promise<{ blob: Blob; fileName: string }> {
  const token = getToken();
  const res = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) {
    throw new Error("Unable to load resume");
  }
  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = /filename="?([^";]+)"?/i.exec(disposition);
  const fileName = match?.[1]?.trim() || "resume";
  return { blob, fileName };
}

/** Open the uploaded resume in a new tab (inline for PDFs when the browser supports it). */
export async function viewResumeFile(url: string): Promise<void> {
  const { blob } = await fetchResumeBlob(url);
  const objectUrl = URL.createObjectURL(blob);
  window.open(objectUrl, "_blank", "noopener,noreferrer");
  window.setTimeout(() => URL.revokeObjectURL(objectUrl), 60_000);
}

/** Create an object URL for in-drawer PDF/text preview. Caller must revoke. */
export async function createResumeObjectUrl(url: string): Promise<{ objectUrl: string; mime: string }> {
  const { blob } = await fetchResumeBlob(url);
  return { objectUrl: URL.createObjectURL(blob), mime: blob.type || "application/octet-stream" };
}

/** Download the original uploaded resume file. */
export async function downloadResumeFile(url: string, preferredName?: string): Promise<void> {
  const downloadURL = url.includes("?") ? `${url}&download=1` : `${url}?download=1`;
  const { blob, fileName } = await fetchResumeBlob(downloadURL);
  const objectUrl = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = objectUrl;
  anchor.download = preferredName?.trim() || fileName || "resume";
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.setTimeout(() => URL.revokeObjectURL(objectUrl), 60_000);
}
