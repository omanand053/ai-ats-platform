"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type AppNotification,
} from "@/services/enterprise.service";

export function NotificationBell() {
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<AppNotification[]>([]);
  const [unread, setUnread] = useState(0);
  const [loading, setLoading] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  const refresh = useCallback(async () => {
    try {
      setLoading(true);
      const data = await listNotifications();
      setItems(data.notifications ?? []);
      setUnread(data.unread ?? 0);
    } catch {
      /* ignore when unauthenticated / offline */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), 45_000);
    return () => window.clearInterval(id);
  }, [refresh]);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    window.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);

  async function onOpen() {
    setOpen((v) => !v);
    if (!open) await refresh();
  }

  async function onRead(id: string) {
    await markNotificationRead(id);
    setItems((prev) =>
      prev.map((n) => (n.id === id ? { ...n, read_at: new Date().toISOString() } : n)),
    );
    setUnread((u) => Math.max(0, u - 1));
  }

  async function onReadAll() {
    await markAllNotificationsRead();
    setItems((prev) => prev.map((n) => ({ ...n, read_at: n.read_at || new Date().toISOString() })));
    setUnread(0);
  }

  return (
    <div className="relative" ref={rootRef}>
      <button
        type="button"
        aria-label={unread ? `${unread} unread notifications` : "Notifications"}
        aria-expanded={open}
        aria-haspopup="dialog"
        className="relative inline-flex h-9 w-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600 transition hover:bg-slate-50 hover:text-slate-900 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800"
        onClick={() => void onOpen()}
      >
        <svg viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M15 17h5l-1.4-1.4A2 2 0 0118 14.2V11a6 6 0 10-12 0v3.2c0 .5-.2 1-.6 1.4L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
          />
        </svg>
        {unread > 0 ? (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-blue-600 px-1 text-[10px] font-bold text-white">
            {unread > 99 ? "99+" : unread}
          </span>
        ) : null}
      </button>

      {open ? (
        <div
          role="dialog"
          aria-label="Notifications"
          className="absolute right-0 z-50 mt-2 w-[min(22rem,calc(100vw-2rem))] overflow-hidden rounded-xl border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-900"
        >
          <div className="flex items-center justify-between border-b border-slate-100 px-3 py-2 dark:border-slate-800">
            <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">Notifications</p>
            <button
              type="button"
              className="text-xs font-medium text-blue-600 hover:underline disabled:opacity-40"
              disabled={!unread}
              onClick={() => void onReadAll()}
            >
              Mark all read
            </button>
          </div>
          <ul className="max-h-80 overflow-auto">
            {loading && !items.length ? (
              <li className="px-3 py-6 text-center text-sm text-slate-500">Loading…</li>
            ) : null}
            {!loading && !items.length ? (
              <li className="px-3 py-6 text-center text-sm text-slate-500">You&apos;re all caught up</li>
            ) : null}
            {items.map((n) => (
              <li
                key={n.id}
                className={`border-b border-slate-50 px-3 py-2.5 last:border-0 dark:border-slate-800 ${
                  n.read_at ? "opacity-70" : "bg-blue-50/40 dark:bg-blue-950/30"
                }`}
              >
                <button
                  type="button"
                  className="w-full text-left"
                  onClick={() => void onRead(n.id)}
                >
                  <p className="text-sm font-medium text-slate-900 dark:text-slate-100">{n.title}</p>
                  {n.body ? <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">{n.body}</p> : null}
                  <p className="mt-1 text-[11px] text-slate-400">{new Date(n.created_at).toLocaleString()}</p>
                </button>
                {n.entity_type === "candidate" && n.entity_id ? (
                  <Link
                    href="/dashboard/candidates"
                    className="mt-1 inline-block text-[11px] font-medium text-blue-600 hover:underline"
                    onClick={() => setOpen(false)}
                  >
                    View candidates
                  </Link>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
