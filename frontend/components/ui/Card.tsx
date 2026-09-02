import type { ReactNode } from "react";

interface CardProps {
  children: ReactNode;
  className?: string;
  hover?: boolean;
  padding?: "none" | "sm" | "md" | "lg";
  header?: ReactNode;
  footer?: ReactNode;
}

const paddings = {
  none: "",
  sm: "p-4",
  md: "p-5",
  lg: "p-5 sm:p-6",
};

export function Card({
  children,
  className = "",
  hover = false,
  padding = "md",
  header,
  footer,
}: CardProps) {
  return (
    <div className={`ats-card ${hover ? "ats-card-hover" : ""} ${className}`}>
      {header ? (
        <div className="border-b border-[var(--border)] px-5 py-3.5 sm:px-6">{header}</div>
      ) : null}
      <div className={paddings[padding]}>{children}</div>
      {footer ? (
        <div className="border-t border-[var(--border)] px-5 py-3 sm:px-6">{footer}</div>
      ) : null}
    </div>
  );
}
