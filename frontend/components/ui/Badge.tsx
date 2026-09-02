import type { ReactNode } from "react";

type BadgeTone = "neutral" | "accent" | "success" | "warning" | "danger";

interface BadgeProps {
  children: ReactNode;
  tone?: BadgeTone;
  className?: string;
}

const tones: Record<BadgeTone, string> = {
  neutral: "ats-badge-neutral",
  accent: "ats-badge-accent",
  success: "ats-badge-success",
  warning: "ats-badge-warning",
  danger: "ats-badge-danger",
};

export function Badge({ children, tone = "neutral", className = "" }: BadgeProps) {
  return <span className={`ats-badge ${tones[tone]} ${className}`}>{children}</span>;
}
