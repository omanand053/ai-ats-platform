"use client";

import type { ButtonHTMLAttributes } from "react";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  loading?: boolean;
  variant?: "primary" | "secondary" | "danger" | "ghost";
  size?: "sm" | "md";
}

const variants = {
  primary:
    "bg-[var(--accent)] text-white shadow-sm hover:bg-[var(--accent-hover)] active:brightness-95 focus-visible:ring-[var(--accent)]",
  secondary:
    "bg-[var(--surface)] text-[var(--text-secondary)] ring-1 ring-[var(--border-strong)] hover:bg-[var(--surface-muted)] active:bg-[var(--surface-muted)] focus-visible:ring-[var(--accent)]",
  danger:
    "bg-[var(--danger)] text-white shadow-sm hover:brightness-110 active:brightness-95 focus-visible:ring-[var(--danger)]",
  ghost:
    "bg-transparent text-[var(--text-secondary)] hover:bg-[var(--surface-muted)] hover:text-[var(--text-primary)] focus-visible:ring-[var(--accent)]",
};

const sizes = {
  sm: "h-9 px-3 text-xs",
  md: "h-10 px-4 text-sm",
};

export function Button({
  loading,
  variant = "primary",
  size = "md",
  className = "",
  children,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={`inline-flex w-full items-center justify-center gap-2 rounded-[var(--radius-md)] font-semibold transition duration-150 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--background)] disabled:cursor-not-allowed disabled:opacity-55 ${variants[variant]} ${sizes[size]} ${className}`}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...props}
    >
      {loading ? (
        <>
          <span
            className={`h-3.5 w-3.5 animate-spin rounded-full border-2 ${
              variant === "secondary" || variant === "ghost"
                ? "border-[var(--border-strong)] border-t-[var(--text-secondary)]"
                : "border-white/40 border-t-white"
            }`}
            aria-hidden
          />
          <span>Please wait…</span>
        </>
      ) : (
        children
      )}
    </button>
  );
}
