"use client";

import {
  cloneElement,
  isValidElement,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type ReactElement,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";

type TooltipProps = {
  label: string;
  children: ReactElement;
  delayMs?: number;
  side?: "top" | "bottom";
};

/** Isolated portal tooltip — only one visible per trigger; never overlaps siblings. */
export function Tooltip({ label, children, delayMs = 280, side = "top" }: TooltipProps) {
  const tipId = useId();
  const triggerRef = useRef<HTMLElement | null>(null);
  const showTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const hideTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [open, setOpen] = useState(false);
  const [coords, setCoords] = useState({ top: 0, left: 0 });

  const clearTimers = useCallback(() => {
    if (showTimer.current) clearTimeout(showTimer.current);
    if (hideTimer.current) clearTimeout(hideTimer.current);
    showTimer.current = null;
    hideTimer.current = null;
  }, []);

  const updatePosition = useCallback(() => {
    const el = triggerRef.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const left = rect.left + rect.width / 2;
    const top = side === "top" ? rect.top - 8 : rect.bottom + 8;
    setCoords({ top, left });
  }, [side]);

  const show = useCallback(() => {
    clearTimers();
    showTimer.current = setTimeout(() => {
      updatePosition();
      setOpen(true);
    }, delayMs);
  }, [clearTimers, delayMs, updatePosition]);

  const hide = useCallback(() => {
    clearTimers();
    hideTimer.current = setTimeout(() => setOpen(false), 80);
  }, [clearTimers]);

  useEffect(() => {
    if (!open) return;
    const onScroll = () => updatePosition();
    window.addEventListener("scroll", onScroll, true);
    window.addEventListener("resize", onScroll);
    return () => {
      window.removeEventListener("scroll", onScroll, true);
      window.removeEventListener("resize", onScroll);
    };
  }, [open, updatePosition]);

  useEffect(() => () => clearTimers(), [clearTimers]);

  if (!isValidElement(children)) {
    return <>{children as ReactNode}</>;
  }

  const child = children as ReactElement<{
    ref?: React.Ref<HTMLElement>;
    onMouseEnter?: (e: React.MouseEvent) => void;
    onMouseLeave?: (e: React.MouseEvent) => void;
    onFocus?: (e: React.FocusEvent) => void;
    onBlur?: (e: React.FocusEvent) => void;
    onTouchStart?: (e: React.TouchEvent) => void;
    "aria-describedby"?: string;
  }>;

  const trigger = cloneElement(child, {
    ref: (node: HTMLElement | null) => {
      triggerRef.current = node;
      const existing = (child as { ref?: React.Ref<HTMLElement> }).ref;
      if (typeof existing === "function") existing(node);
      else if (existing && typeof existing === "object") {
        (existing as { current: HTMLElement | null }).current = node;
      }
    },
    "aria-describedby": open ? tipId : undefined,
    onMouseEnter: (e: React.MouseEvent) => {
      child.props.onMouseEnter?.(e);
      show();
    },
    onMouseLeave: (e: React.MouseEvent) => {
      child.props.onMouseLeave?.(e);
      hide();
    },
    onFocus: (e: React.FocusEvent) => {
      child.props.onFocus?.(e);
      show();
    },
    onBlur: (e: React.FocusEvent) => {
      child.props.onBlur?.(e);
      hide();
    },
    onTouchStart: (e: React.TouchEvent) => {
      child.props.onTouchStart?.(e);
      show();
      hideTimer.current = setTimeout(() => setOpen(false), 1800);
    },
  });

  return (
    <>
      {trigger}
      {open &&
        typeof document !== "undefined" &&
        createPortal(
          <span
            id={tipId}
            role="tooltip"
            className="pointer-events-none fixed z-[1000] -translate-x-1/2 whitespace-nowrap rounded-md bg-slate-900 px-2.5 py-1 text-[11px] font-medium text-white shadow-lg"
            style={{
              top: coords.top,
              left: coords.left,
              transform:
                side === "top"
                  ? "translate(-50%, -100%)"
                  : "translate(-50%, 0)",
            }}
          >
            {label}
          </span>,
          document.body,
        )}
    </>
  );
}
