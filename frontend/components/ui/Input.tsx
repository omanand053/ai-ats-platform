import type { InputHTMLAttributes, ReactNode } from "react";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  hint?: string;
  optional?: boolean;
}

export function Input({
  label,
  error,
  hint,
  optional,
  id,
  className = "",
  ...props
}: InputProps) {
  const inputId = id ?? props.name;
  const hintId = hint ? `${inputId}-hint` : undefined;
  const errorId = error ? `${inputId}-error` : undefined;

  return (
    <div className="space-y-1.5">
      <div className="flex items-baseline justify-between gap-2">
        <label htmlFor={inputId} className="block text-sm font-medium text-[var(--text-secondary)]">
          {label}
        </label>
        {optional ? <span className="text-xs text-[var(--text-muted)]">Optional</span> : null}
      </div>
      <input
        id={inputId}
        className={`ats-input ${error ? "!border-[var(--danger)] focus:!shadow-[0_0_0_3px_color-mix(in_srgb,var(--danger)_18%,transparent)]" : ""} ${className}`}
        aria-invalid={error ? true : undefined}
        aria-describedby={[hintId, errorId].filter(Boolean).join(" ") || undefined}
        {...props}
      />
      {hint && !error ? (
        <p id={hintId} className="text-xs text-[var(--text-muted)]">
          {hint}
        </p>
      ) : null}
      {error ? (
        <p id={errorId} className="text-xs font-medium text-[var(--danger)]" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}

interface FormSectionProps {
  title: string;
  description?: string;
  children: ReactNode;
}

export function FormSection({ title, description, children }: FormSectionProps) {
  return (
    <section className="ats-card space-y-4 p-5 sm:p-6">
      <div className="border-b border-[var(--border)] pb-3">
        <h3 className="ats-section-title">{title}</h3>
        {description ? <p className="mt-1 text-sm text-[var(--text-muted)]">{description}</p> : null}
      </div>
      <div className="space-y-4">{children}</div>
    </section>
  );
}
