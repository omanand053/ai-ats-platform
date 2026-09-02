export const container = "mx-auto w-full max-w-7xl";
export const pagePadding = "px-4 py-6 sm:px-6 sm:py-8 lg:px-8";

/** Semantic Tailwind helpers bound to CSS design tokens. */
export const tokens = {
  surface: "bg-[var(--surface)]",
  surfaceAlt: "bg-[var(--surface-muted)]",
  cardBorder: "border-[var(--border)]",
  muted: "text-[var(--text-muted)]",
  primaryText: "text-[var(--text-primary)]",
  accent: "text-[var(--accent)]",
  shadowSm: "shadow-[var(--shadow-sm)]",
};

export const ds = {
  card: `rounded-[var(--radius-lg)] border ${tokens.cardBorder} ${tokens.surface} ${tokens.shadowSm}`,
  kpiCard: `rounded-[var(--radius-lg)] border ${tokens.cardBorder} bg-[var(--surface)] px-4 py-4 ${tokens.shadowSm}`,
  panel: `rounded-[var(--radius-lg)] border ${tokens.cardBorder} bg-[var(--surface)] ${tokens.shadowSm}`,
  small: "text-sm",
  heading: "text-[1.25rem] font-semibold leading-tight tracking-tight",
  listRow:
    "flex items-center justify-between gap-3 rounded-[var(--radius-md)] border border-[var(--border)] bg-[var(--surface)] px-3.5 py-2.5 transition hover:border-[var(--border-strong)] hover:bg-[var(--surface-muted)]",
};

export function classNames(...list: Array<string | false | null | undefined>) {
  return list.filter(Boolean).join(" ");
}
