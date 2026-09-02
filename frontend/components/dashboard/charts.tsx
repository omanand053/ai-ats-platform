"use client";

import React, { useState, useRef, useEffect } from "react";
import { ds, tokens, classNames } from "./designSystem";

export function ChartEmpty({ message = "No data yet — add jobs and candidates to populate this chart." }: { message?: string }) {
  return (
    <div className={classNames(ds.card, "min-h-[8rem] border-dashed px-4 py-6 text-center")} role="status">
      <p className="max-w-xs text-sm leading-5 text-[#8b877f]">{message}</p>
    </div>
  );
}

export function BarChart({
  items,
  height = 180,
  emptyMessage,
}: {
  items: { label: string; value: number }[];
  height?: number;
  emptyMessage?: string;
}) {
  const max = Math.max(1, ...items.map((i) => i.value));
  if (!items.length) return <ChartEmpty message={emptyMessage} />;

  return (
    <div className="space-y-2.5" style={{ minHeight: height }} role="img" aria-label="Bar chart">
      {items.map((item, idx) => (
        <div key={`${item.label}-${idx}`} className="grid grid-cols-[minmax(4.5rem,7rem)_1fr_2.5rem] items-center gap-2">
          <span className="truncate text-xs text-[#6b7280]" title={item.label}>
            {item.label}
          </span>
          <div className="h-2.5 overflow-hidden rounded-full bg-gray-100">
            <div
              className="h-full rounded-full bg-gradient-to-r from-[var(--brand)] to-[color:var(--brand-hover)] transition-all duration-500 ease-out"
              style={{ width: `${(item.value / max) * 100}%` }}
            />
          </div>
          <span className="text-right text-xs font-semibold tabular-nums text-[#0b1220]">
            {item.value}
          </span>
        </div>
      ))}
    </div>
  );
}

export function FunnelChart({
  items,
  emptyMessage,
}: {
  items: { label: string; value: number }[];
  emptyMessage?: string;
}) {
  if (!items.length) return <ChartEmpty message={emptyMessage} />;
  const max = Math.max(1, ...items.map((i) => i.value));
  return (
    <div className="space-y-2" role="img" aria-label="Hiring funnel">
      {items.map((item, idx) => {
        const width = 40 + (item.value / max) * 60;
        return (
          <div key={`${item.label}-${idx}`} className="flex flex-col items-center">
            <div
              className="rounded-[12px] border border-[var(--brand-soft)] bg-gradient-to-r from-[var(--brand-soft)] to-white px-3 py-2 text-center text-xs font-semibold text-[var(--text-primary)] shadow-sm transition"
              style={{ width: `${width}%`, minWidth: "40%" }}
            >
              <span className="text-sm font-medium text-[var(--text-primary)]">{item.label}</span>
              <span className="ml-2 inline-block rounded-full bg-[var(--brand-soft)] px-2 py-0.5 text-xs font-semibold text-[var(--brand)]">{item.value}</span>
            </div>
            {idx < items.length - 1 && <div className="h-2 w-px bg-gray-100" aria-hidden />}
          </div>
        );
      })}
    </div>
  );
}

export function TrendSparkline({
  points,
  emptyMessage,
}: {
  points: { period: string; count: number }[];
  emptyMessage?: string;
}) {
  if (!points.length) return <ChartEmpty message={emptyMessage} />;
  const max = Math.max(1, ...points.map((p) => p.count));
  const w = 320;
  const h = 80;
  const step = points.length > 1 ? w / (points.length - 1) : w;
  const coords = points.map((p, i) => {
    const x = i * step;
    const y = h - (p.count / max) * (h - 12) - 6;
    return { x, y };
  });

  const d = coords.map((c, i) => `${i === 0 ? "M" : "L"}${c.x},${c.y}`).join(" ");
  const areaD = `${d} L ${w},${h} L 0,${h} Z`;

  return (
    <div role="img" aria-label="Trend chart">
      <svg viewBox={`0 0 ${w} ${h}`} className="h-28 w-full overflow-visible" aria-hidden>
        <defs>
          <linearGradient id="ts-grad" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor="var(--brand-soft)" stopOpacity="0.95" />
            <stop offset="100%" stopColor="#ffffff" stopOpacity="0.06" />
          </linearGradient>
        </defs>
        <path d={areaD} fill="url(#ts-grad)" stroke="none" />
        <path d={d} fill="none" stroke="var(--brand)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
        {coords.map((c, i) => (
          <circle key={i} cx={c.x} cy={c.y} r={3.2} fill="#fff" stroke="var(--brand)" strokeWidth={1.4} />
        ))}
      </svg>
      <div className="mt-2 flex items-center justify-between text-[11px] text-[var(--text-muted)]">
        <span>{points[0]?.period}</span>
        <span>{points[points.length - 1]?.period}</span>
      </div>
    </div>
  );
}

function formatPercent(value: number, total: number) {
  if (total === 0) return "0%";
  return `${Math.round((value / total) * 100)}%`;
}

export function DonutChart({ items, size = 180 }: { items: { label: string; value: number; color?: string }[]; size?: number }) {
  const total = items.reduce((s, it) => s + Math.max(0, it.value), 0);
  const radius = size / 2 - 14;
  const circumference = 2 * Math.PI * radius;
  let offset = 0;
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [hover, setHover] = useState<{ label: string; value: number; percent: string; x: number; y: number } | null>(null);

  useEffect(() => {
    function onLeave() {
      setHover(null);
    }
    const el = containerRef.current;
    if (!el) return;
    el.addEventListener("mouseleave", onLeave);
    return () => el.removeEventListener("mouseleave", onLeave);
  }, []);

  return (
    <div ref={containerRef} className="relative flex items-center gap-4">
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} className="block">
        <g transform={`translate(${size / 2},${size / 2})`}>
          {items.map((it, i) => {
            const val = Math.max(0, it.value);
            const len = (val / Math.max(1, total)) * circumference;
            const strokeDasharray = `${len} ${circumference - len}`;
            const color = it.color ?? (i === 0 ? "var(--brand)" : `hsl(${(i * 55) % 360} 72% 52%)`);
            const idx = i;
            const circle = (
              <circle
                key={i}
                r={radius}
                cx={0}
                cy={0}
                fill="transparent"
                stroke={color}
                strokeWidth={hover?.label === it.label ? 16 : 12}
                strokeDasharray={strokeDasharray}
                strokeDashoffset={-offset}
                strokeLinecap="round"
                transform="rotate(-90)"
                style={{ transition: "stroke-dashoffset 0.6s, stroke-width 0.18s, opacity 0.22s" }}
                onMouseEnter={(e) => {
                  const rect = containerRef.current?.getBoundingClientRect();
                  const x = rect ? e.clientX - rect.left : 0;
                  const y = rect ? e.clientY - rect.top : 0;
                  setHover({ label: it.label, value: it.value, percent: formatPercent(it.value, total), x, y });
                }}
                onMouseMove={(e) => {
                  const rect = containerRef.current?.getBoundingClientRect();
                  const x = rect ? e.clientX - rect.left : 0;
                  const y = rect ? e.clientY - rect.top : 0;
                  setHover((s) => (s && s.label === it.label ? { ...s, x, y } : s));
                }}
              />
            );
            offset += len;
            return circle;
          })}
          <circle r={radius - 20} fill="var(--surface)" />
          <text x="0" y="0" textAnchor="middle" dominantBaseline="central" className="text-sm font-semibold text-[var(--text-primary)]" style={{ fontSize: 14 }}>
            {total}
          </text>
        </g>
      </svg>

      <div className="flex flex-col text-sm">
        {items.map((it, i) => (
          <div
            key={i}
            className="flex items-center gap-3 py-1 cursor-default transition-colors"
            onMouseEnter={() => {
              const el = containerRef.current;
              if (!el) return;
              const rect = el.getBoundingClientRect();
              setHover({ label: it.label, value: it.value, percent: formatPercent(it.value, total), x: rect.width / 2, y: rect.height / 2 });
            }}
            onMouseLeave={() => setHover(null)}
          >
            <span style={{ background: it.color ?? (i === 0 ? "var(--brand)" : `hsl(${(i * 55) % 360} 72% 52%)`) }} className="inline-block h-3 w-3 rounded-full" />
            <div className="flex items-baseline gap-2">
              <span className="text-[13px] text-[var(--text-primary)]">{it.label}</span>
              <span className="text-xs text-[var(--text-muted)]">{it.value}</span>
              <span className="ml-1 text-xs text-[var(--text-muted)]">{formatPercent(it.value, total)}</span>
            </div>
          </div>
        ))}
      </div>

      {hover ? (
        <div
          className="pointer-events-none absolute z-30 w-max rounded-md bg-[var(--surface)]/95 px-3 py-1 text-sm shadow-lg"
          style={{ left: hover.x + 8, top: hover.y + 8 }}
        >
          <div className="font-medium text-[var(--text-primary)]">{hover.label}</div>
          <div className="text-xs text-[var(--text-muted)]">{hover.value} • {hover.percent}</div>
        </div>
      ) : null}
    </div>
  );
}

export function PieChart({ items, size = 160 }: { items: { label: string; value: number; color?: string }[]; size?: number }) {
  const total = items.reduce((s, it) => s + Math.max(0, it.value), 0);
  const radius = size / 2;
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [hover, setHover] = useState<{ label: string; value: number; percent: string; x: number; y: number } | null>(null);
  let cumulative = 0;

  useEffect(() => {
    function onLeave() {
      setHover(null);
    }
    const el = containerRef.current;
    if (!el) return;
    el.addEventListener("mouseleave", onLeave);
    return () => el.removeEventListener("mouseleave", onLeave);
  }, []);

  return (
    <div ref={containerRef} className="relative flex items-center gap-4">
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <g transform={`translate(${radius},${radius})`}>
          {items.map((it, idx) => {
            const value = Math.max(0, it.value);
            const startAngle = (cumulative / Math.max(1, total)) * Math.PI * 2;
            const endAngle = ((cumulative + value) / Math.max(1, total)) * Math.PI * 2;
            cumulative += value;
            const x1 = Math.cos(startAngle) * radius;
            const y1 = Math.sin(startAngle) * radius;
            const x2 = Math.cos(endAngle) * radius;
            const y2 = Math.sin(endAngle) * radius;
            const largeArc = endAngle - startAngle > Math.PI ? 1 : 0;
            const d = `M 0 0 L ${x1} ${y1} A ${radius} ${radius} 0 ${largeArc} 1 ${x2} ${y2} Z`;
            const color = it.color ?? `hsl(${(idx * 55) % 360} 72% 52%)`;
            return (
              <path
                key={idx}
                d={d}
                fill={color}
                stroke="#fff"
                strokeWidth={1}
                style={{ transition: "transform 0.28s ease, opacity 0.18s" }}
                onMouseEnter={(e) => {
                  const rect = containerRef.current?.getBoundingClientRect();
                  const x = rect ? e.clientX - rect.left : 0;
                  const y = rect ? e.clientY - rect.top : 0;
                  setHover({ label: it.label, value: it.value, percent: formatPercent(it.value, total), x, y });
                }}
                onMouseMove={(e) => {
                  const rect = containerRef.current?.getBoundingClientRect();
                  const x = rect ? e.clientX - rect.left : 0;
                  const y = rect ? e.clientY - rect.top : 0;
                  setHover((s) => (s && s.label === it.label ? { ...s, x, y } : s));
                }}
              />
            );
          })}
        </g>
      </svg>

      <div className="flex flex-col text-sm">
        {items.map((it, i) => (
          <div
            key={i}
            className="flex items-center gap-3 py-1 cursor-default transition-colors"
            onMouseEnter={() => {
              const el = containerRef.current;
              if (!el) return;
              const rect = el.getBoundingClientRect();
              setHover({ label: it.label, value: it.value, percent: formatPercent(it.value, total), x: rect.width / 2, y: rect.height / 2 });
            }}
            onMouseLeave={() => setHover(null)}
          >
            <span style={{ background: it.color ?? `hsl(${(i * 55) % 360} 72% 52%)` }} className="inline-block h-3 w-3 rounded-full" />
            <div className="flex items-baseline gap-2">
              <span className="text-[13px] text-[var(--text-primary)]">{it.label}</span>
              <span className="text-xs text-[var(--text-muted)]">{it.value}</span>
              <span className="ml-1 text-xs text-[var(--text-muted)]">{formatPercent(it.value, total)}</span>
            </div>
          </div>
        ))}
      </div>

      {hover ? (
        <div
          className="pointer-events-none absolute z-30 w-max rounded-md bg-[var(--surface)]/95 px-3 py-1 text-sm shadow-lg"
          style={{ left: hover.x + 8, top: hover.y + 8 }}
        >
          <div className="font-medium text-[var(--text-primary)]">{hover.label}</div>
          <div className="text-xs text-[var(--text-muted)]">{hover.value} • {hover.percent}</div>
        </div>
      ) : null}
    </div>
  );
}
