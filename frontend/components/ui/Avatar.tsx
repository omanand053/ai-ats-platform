export function Avatar({ name, size = "md" }: { name: string; size?: "sm" | "md" | "lg" }) {
  const initials = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase() ?? "")
    .join("") || "?";
  const sizes = {
    sm: "h-8 w-8 text-xs",
    md: "h-10 w-10 text-sm",
    lg: "h-12 w-12 text-base",
  };
  const hue =
    Array.from(name).reduce((acc, ch) => acc + ch.charCodeAt(0), 0) % 360;
  return (
    <span
      className={`inline-flex shrink-0 items-center justify-center rounded-full font-semibold text-white ${sizes[size]}`}
      style={{ background: `hsl(${hue} 42% 42%)` }}
      aria-hidden
    >
      {initials}
    </span>
  );
}
