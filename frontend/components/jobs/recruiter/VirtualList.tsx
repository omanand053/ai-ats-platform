"use client";

import { useEffect, useMemo, useRef, useState } from "react";

const DEFAULT_ITEM_HEIGHT = 92;
const OVERSCAN = 8;

export function VirtualList<T>({
  items,
  height = 560,
  estimateSize = DEFAULT_ITEM_HEIGHT,
  renderItem,
  className = "",
}: {
  items: T[];
  height?: number;
  estimateSize?: number;
  renderItem: (item: T, index: number) => React.ReactNode;
  className?: string;
}) {
  const scrollerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);

  useEffect(() => {
    const el = scrollerRef.current;
    if (!el) return;
    const onScroll = () => setScrollTop(el.scrollTop);
    el.addEventListener("scroll", onScroll, { passive: true });
    return () => el.removeEventListener("scroll", onScroll);
  }, []);

  const { start, end, offsetY, totalHeight } = useMemo(() => {
    const total = items.length * estimateSize;
    const startIdx = Math.max(0, Math.floor(scrollTop / estimateSize) - OVERSCAN);
    const visible = Math.ceil(height / estimateSize) + OVERSCAN * 2;
    const endIdx = Math.min(items.length, startIdx + visible);
    return {
      start: startIdx,
      end: endIdx,
      offsetY: startIdx * estimateSize,
      totalHeight: total,
    };
  }, [items.length, estimateSize, height, scrollTop]);

  const slice = items.slice(start, end);

  return (
    <div
      ref={scrollerRef}
      className={`overflow-auto ${className}`}
      style={{ height }}
      role="list"
    >
      <div style={{ height: totalHeight, position: "relative" }}>
        <div style={{ transform: `translateY(${offsetY}px)` }}>
          {slice.map((item, i) => (
            <div key={start + i} role="listitem" style={{ minHeight: estimateSize }}>
              {renderItem(item, start + i)}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
