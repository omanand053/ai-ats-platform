"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { PageHeader } from "@/components/ui/PageHeader";
import { Spinner } from "@/components/ui/Spinner";
import { ToastContainer } from "@/components/ui/Toast";
import { useCurrentUser } from "@/hooks/useCurrentUser";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import { getAISettings, updateAISettings, type CompanyAISettings } from "@/services/enterprise.service";

function WeightSlider({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <label className="block text-sm text-[#0b1220]">
      <div className="mb-1 flex justify-between">
        <span className="font-medium">{label}</span>
        <span className="tabular-nums text-[#6b7280]">{Math.round(value * 100)}%</span>
      </div>
      <input
        type="range"
        min={0}
        max={100}
        step={1}
        value={Math.round(value * 100)}
        onChange={(e) => onChange(Number(e.target.value) / 100)}
        className="w-full"
      />
    </label>
  );
}

export function AISettingsView() {
  const ready = useRequireAuth();
  const router = useRouter();
  const { isAdmin, loading: userLoading } = useCurrentUser();
  const { toasts, dismiss, show } = useToast();
  const [settings, setSettings] = useState<CompanyAISettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!ready || userLoading) return;
    if (!isAdmin) {
      setLoading(false);
      router.replace("/dashboard");
      return;
    }
    getAISettings()
      .then(setSettings)
      .catch((err) => show(err instanceof ApiClientError ? err.message : "Failed to load settings", "error"))
      .finally(() => setLoading(false));
  }, [ready, userLoading, isAdmin, router, show]);

  async function save() {
    if (!settings) return;
    setSaving(true);
    try {
      const updated = await updateAISettings(settings);
      setSettings(updated);
      show("AI settings saved — applied on next semantic ranking");
    } catch (err) {
      show(err instanceof ApiClientError ? err.message : "Failed to save configuration", "error");
    } finally {
      setSaving(false);
    }
  }

  if (!ready || userLoading || !isAdmin || loading) {
    return (
      <>
        <Spinner label="Loading AI settings..." />
      </>
    );
  }

  if (!settings) {
    return (
      <>
        <ErrorState
          title="AI settings unavailable"
          description="We couldn’t load configuration for this company."
          onRetry={() => window.location.reload()}
        />
      </>
    );
  }

  return (
    <>
      <>
        <PageHeader
          title="AI Configuration"
          subtitle="Admin-configurable Overall AI Match weights and thresholds. No code deploy required."
        />
        <Card className="mt-4 max-w-2xl space-y-4" padding="lg">
          <WeightSlider
            label="Semantic similarity"
            value={settings.weight_semantic}
            onChange={(v) => setSettings({ ...settings, weight_semantic: v })}
          />
          <WeightSlider
            label="Skills"
            value={settings.weight_skills}
            onChange={(v) => setSettings({ ...settings, weight_skills: v })}
          />
          <WeightSlider
            label="Experience"
            value={settings.weight_experience}
            onChange={(v) => setSettings({ ...settings, weight_experience: v })}
          />
          <WeightSlider
            label="Education"
            value={settings.weight_education}
            onChange={(v) => setSettings({ ...settings, weight_education: v })}
          />
          <WeightSlider
            label="Projects"
            value={settings.weight_projects}
            onChange={(v) => setSettings({ ...settings, weight_projects: v })}
          />
          <label className="block text-sm">
            <span className="font-medium text-[#0b1220]">
              Confidence threshold ({Math.round(settings.confidence_threshold)})
            </span>
            <input
              type="range"
              min={0}
              max={100}
              className="mt-2 w-full"
              value={settings.confidence_threshold}
              onChange={(e) =>
                setSettings({ ...settings, confidence_threshold: Number(e.target.value) })
              }
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium text-[#0b1220]">
              Eligibility threshold ({Math.round(settings.eligibility_threshold)})
            </span>
            <input
              type="range"
              min={0}
              max={100}
              className="mt-2 w-full"
              value={settings.eligibility_threshold}
              onChange={(e) =>
                setSettings({ ...settings, eligibility_threshold: Number(e.target.value) })
              }
            />
          </label>
          <Button className="w-auto px-5" loading={saving} onClick={save}>
            Save configuration
          </Button>
          <p className="text-xs text-[#6b7280]">Weights are normalized to 100% on save. Admin role required to update.</p>
        </Card>
      </>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
