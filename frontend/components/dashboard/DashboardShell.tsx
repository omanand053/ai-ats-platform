"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { NotificationBell } from "@/components/dashboard/NotificationBell";
import { container, pagePadding } from "./designSystem";
import { Button } from "@/components/ui/Button";
import { Avatar } from "@/components/ui/Avatar";
import FloatingAssistant from "@/components/common/FloatingAssistant";
import { useCurrentUser } from "@/hooks/useCurrentUser";
import { logout } from "@/services/auth.service";

const SIDEBAR_KEY = "ats-sidebar-collapsed";

type NavItem = {
  href: string;
  label: string;
  icon: (props: { className?: string }) => ReactNode;
  adminOnly?: boolean;
  section?: "main" | "insights" | "admin";
};

const baseNav: NavItem[] = [
  { href: "/dashboard", label: "Overview", icon: OverviewIcon, section: "main" },
  { href: "/dashboard/jobs", label: "Jobs", icon: JobsIcon, section: "main" },
  { href: "/dashboard/candidates", label: "Candidates", icon: CandidatesIcon, section: "main" },
  { href: "/dashboard/calendar", label: "Calendar", icon: CalendarIcon, section: "insights" },
  { href: "/dashboard/reports", label: "Reports", icon: ReportsIcon, section: "insights" },
  { href: "/dashboard/settings/ai", label: "AI Settings", icon: SettingsIcon, adminOnly: true, section: "admin" },
];

function pageMeta(pathname: string): { title: string; crumbs: { label: string; href?: string }[] } {
  if (pathname === "/dashboard") {
    return { title: "Overview", crumbs: [{ label: "Workspace" }, { label: "Overview" }] };
  }
  if (pathname.startsWith("/dashboard/jobs/new")) {
    return {
      title: "New job",
      crumbs: [{ label: "Jobs", href: "/dashboard/jobs" }, { label: "Create" }],
    };
  }
  if (pathname.match(/\/dashboard\/jobs\/[^/]+\/edit/)) {
    return {
      title: "Edit job",
      crumbs: [{ label: "Jobs", href: "/dashboard/jobs" }, { label: "Edit" }],
    };
  }
  if (pathname.match(/\/dashboard\/jobs\/[^/]+/)) {
    return {
      title: "Job detail",
      crumbs: [{ label: "Jobs", href: "/dashboard/jobs" }, { label: "Detail" }],
    };
  }
  if (pathname.startsWith("/dashboard/jobs")) {
    return { title: "Jobs", crumbs: [{ label: "Hiring" }, { label: "Jobs" }] };
  }
  if (pathname.startsWith("/dashboard/candidates/new")) {
    return {
      title: "Add candidate",
      crumbs: [{ label: "Candidates", href: "/dashboard/candidates" }, { label: "Create" }],
    };
  }
  if (pathname.match(/\/dashboard\/candidates\/[^/]+\/edit/)) {
    return {
      title: "Edit candidate",
      crumbs: [{ label: "Candidates", href: "/dashboard/candidates" }, { label: "Edit" }],
    };
  }
  if (pathname.match(/\/dashboard\/candidates\/[^/]+/)) {
    return {
      title: "Candidate",
      crumbs: [{ label: "Candidates", href: "/dashboard/candidates" }, { label: "Profile" }],
    };
  }
  if (pathname.startsWith("/dashboard/candidates")) {
    return { title: "Candidates", crumbs: [{ label: "Hiring" }, { label: "Candidates" }] };
  }
  if (pathname.startsWith("/dashboard/calendar")) {
    return { title: "Calendar", crumbs: [{ label: "Insights" }, { label: "Calendar" }] };
  }
  if (pathname.startsWith("/dashboard/reports")) {
    return { title: "Reports", crumbs: [{ label: "Insights" }, { label: "Reports" }] };
  }
  if (pathname.startsWith("/dashboard/settings")) {
    return { title: "AI Settings", crumbs: [{ label: "Admin" }, { label: "AI Settings" }] };
  }
  return { title: "Workspace", crumbs: [{ label: "Dashboard" }] };
}

export function DashboardShell({ children }: { children: ReactNode }) {
  const pathname = usePathname() || "/dashboard";
  const router = useRouter();
  const { isAdmin, user } = useCurrentUser();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [dark, setDark] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);
  const [search, setSearch] = useState("");
  const profileRef = useRef<HTMLDivElement | null>(null);

  const navItems = useMemo(
    () => baseNav.filter((item) => !item.adminOnly || isAdmin),
    [isAdmin],
  );

  const meta = useMemo(() => pageMeta(pathname), [pathname]);

  useEffect(() => {
    setMobileOpen(false);
    setProfileOpen(false);
  }, [pathname]);

  useEffect(() => {
    const storedTheme = window.localStorage.getItem("ats-theme");
    const preferDark =
      storedTheme === "dark" ||
      (!storedTheme && window.matchMedia("(prefers-color-scheme: dark)").matches);
    setDark(preferDark);
    document.documentElement.classList.toggle("dark", preferDark);

    const storedCollapse = window.localStorage.getItem(SIDEBAR_KEY);
    setCollapsed(storedCollapse === "1");
  }, []);

  useEffect(() => {
    if (!mobileOpen && !profileOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setMobileOpen(false);
        setProfileOpen(false);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [mobileOpen, profileOpen]);

  useEffect(() => {
    if (!profileOpen) return;
    const onClick = (e: MouseEvent) => {
      if (profileRef.current && !profileRef.current.contains(e.target as Node)) {
        setProfileOpen(false);
      }
    };
    window.addEventListener("mousedown", onClick);
    return () => window.removeEventListener("mousedown", onClick);
  }, [profileOpen]);

  function toggleTheme() {
    const next = !dark;
    setDark(next);
    document.documentElement.classList.toggle("dark", next);
    window.localStorage.setItem("ats-theme", next ? "dark" : "light");
  }

  function toggleCollapsed() {
    setCollapsed((prev) => {
      const next = !prev;
      window.localStorage.setItem(SIDEBAR_KEY, next ? "1" : "0");
      return next;
    });
  }

  function handleLogout() {
    logout();
    router.push("/login");
  }

  function submitSearch(e: React.FormEvent) {
    e.preventDefault();
    const q = search.trim();
    if (!q) return;
    router.push(`/dashboard/candidates?q=${encodeURIComponent(q)}`);
  }

  const displayName = user
    ? `${user.first_name || ""} ${user.last_name || ""}`.trim() || user.email
    : "Account";
  const roleLabel = user?.role ? user.role.replace(/_/g, " ") : "";

  const sections: { key: NavItem["section"]; label: string }[] = [
    { key: "main", label: "Hiring" },
    { key: "insights", label: "Insights" },
    { key: "admin", label: "Admin" },
  ];

  function renderNav(compact: boolean, onNavigate?: () => void) {
    return sections.map((section) => {
      const items = navItems.filter((n) => n.section === section.key);
      if (items.length === 0) return null;
      return (
        <div key={section.key} className={compact ? "mb-2" : "mb-4"}>
          {!compact ? (
            <p className="mb-1.5 px-3 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--sidebar-muted)]">
              {section.label}
            </p>
          ) : null}
          <div className="flex flex-col gap-0.5">
            {items.map((item) => {
              const active =
                item.href === "/dashboard"
                  ? pathname === "/dashboard"
                  : pathname.startsWith(item.href);
              const Icon = item.icon;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  title={compact ? item.label : undefined}
                  aria-current={active ? "page" : undefined}
                  onClick={onNavigate}
                  className={`group inline-flex items-center gap-2.5 rounded-[var(--radius-md)] px-2.5 py-2 text-sm font-medium transition ${
                    compact ? "justify-center px-0" : ""
                  } ${
                    active
                      ? "bg-white/10 text-white"
                      : "text-[var(--sidebar-muted)] hover:bg-white/[0.06] hover:text-white"
                  }`}
                >
                  <Icon className={`h-4 w-4 shrink-0 ${active ? "text-white" : "opacity-80 group-hover:opacity-100"}`} />
                  {!compact ? <span className="truncate">{item.label}</span> : null}
                </Link>
              );
            })}
          </div>
        </div>
      );
    });
  }

  const sidebarWidth = collapsed ? "var(--sidebar-collapsed-width)" : "var(--sidebar-width)";

  return (
    <div className="min-h-full lg:flex">
      <aside
        className="sticky top-0 hidden h-screen shrink-0 flex-col border-r border-white/5 bg-[var(--sidebar-bg)] lg:flex"
        style={{ width: sidebarWidth, transition: "var(--transition-sidebar)" }}
        aria-label="Primary sidebar"
      >
        <div className={`flex items-center gap-3 px-3 py-4 ${collapsed ? "justify-center" : "px-4"}`}>
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-white/10 text-[11px] font-bold tracking-wide text-white">
            AI
          </div>
          {!collapsed ? (
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-white">AI ATS</p>
              <p className="truncate text-xs text-[var(--sidebar-muted)]">Hiring workspace</p>
            </div>
          ) : null}
        </div>

        <nav className="flex flex-1 flex-col overflow-y-auto px-2.5 pb-3" aria-label="Primary">
          {renderNav(collapsed)}
        </nav>

        <div className="border-t border-white/5 p-2.5">
          <button
            type="button"
            onClick={toggleCollapsed}
            className="inline-flex w-full items-center justify-center gap-2 rounded-[var(--radius-md)] px-2.5 py-2 text-xs font-medium text-[var(--sidebar-muted)] transition hover:bg-white/[0.06] hover:text-white"
            aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            aria-pressed={collapsed}
          >
            <CollapseIcon className={`h-4 w-4 transition ${collapsed ? "rotate-180" : ""}`} />
            {!collapsed ? <span>Collapse</span> : null}
          </button>
        </div>
      </aside>

      {mobileOpen ? (
        <div className="fixed inset-0 z-50 lg:hidden" role="dialog" aria-modal="true" aria-label="Navigation">
          <button
            type="button"
            className="absolute inset-0 bg-slate-900/40 backdrop-blur-[1px]"
            aria-label="Close navigation"
            onClick={() => setMobileOpen(false)}
          />
          <aside className="absolute inset-y-0 left-0 flex w-[min(18rem,86vw)] flex-col bg-[var(--surface)] shadow-[var(--shadow-lg)]">
            <div className="flex items-center justify-between gap-3 border-b border-[var(--border)] px-4 py-3.5">
              <div className="flex items-center gap-2.5">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-[var(--brand)] text-[10px] font-bold text-white dark:bg-white dark:text-slate-900">
                  AI
                </div>
                <span className="text-sm font-semibold text-[var(--text-primary)]">AI ATS</span>
              </div>
              <Button variant="ghost" size="sm" className="w-auto px-2" onClick={() => setMobileOpen(false)}>
                Close
              </Button>
            </div>
            <nav className="flex flex-1 flex-col gap-0.5 overflow-y-auto px-3 py-3" aria-label="Primary mobile">
              {navItems.map((item) => {
                const active =
                  item.href === "/dashboard"
                    ? pathname === "/dashboard"
                    : pathname.startsWith(item.href);
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    aria-current={active ? "page" : undefined}
                    onClick={() => setMobileOpen(false)}
                    className={`inline-flex items-center gap-2.5 rounded-[var(--radius-md)] px-3 py-2.5 text-sm font-medium transition ${
                      active
                        ? "bg-[var(--accent-soft)] text-[var(--accent)]"
                        : "text-[var(--text-secondary)] hover:bg-[var(--surface-muted)]"
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {item.label}
                  </Link>
                );
              })}
            </nav>
            <div className="border-t border-[var(--border)] p-3">
              <Button variant="ghost" size="sm" className="w-full justify-start" onClick={handleLogout}>
                Log out
              </Button>
            </div>
          </aside>
        </div>
      ) : null}

      <div className="flex min-w-0 flex-1 flex-col">
        <header
          className="sticky top-0 z-30 border-b border-[var(--border)] bg-[color-mix(in_srgb,var(--surface)_88%,transparent)] backdrop-blur-md"
          style={{ minHeight: "var(--header-height)" }}
        >
          <div className="flex items-center gap-3 px-4 py-2.5 sm:px-6 lg:px-8">
            <div className="flex min-w-0 flex-1 items-center gap-3">
              <Button
                variant="secondary"
                size="sm"
                className="w-auto shrink-0 px-2.5 lg:hidden"
                aria-label="Open navigation"
                aria-expanded={mobileOpen}
                onClick={() => setMobileOpen(true)}
              >
                <MenuIcon className="h-4 w-4" />
              </Button>

              <div className="min-w-0">
                <nav className="mb-0.5 hidden items-center gap-1.5 text-xs text-[var(--text-muted)] sm:flex" aria-label="Breadcrumb">
                  {meta.crumbs.map((crumb, i) => (
                    <span key={`${crumb.label}-${i}`} className="inline-flex items-center gap-1.5">
                      {i > 0 ? <span aria-hidden className="text-[var(--border-strong)]">/</span> : null}
                      {crumb.href ? (
                        <Link href={crumb.href} className="hover:text-[var(--text-secondary)]">
                          {crumb.label}
                        </Link>
                      ) : (
                        <span className={i === meta.crumbs.length - 1 ? "text-[var(--text-secondary)]" : ""}>
                          {crumb.label}
                        </span>
                      )}
                    </span>
                  ))}
                </nav>
                <h1 className="truncate text-sm font-semibold tracking-tight text-[var(--text-primary)] sm:text-[0.9375rem]">
                  {meta.title}
                </h1>
              </div>
            </div>

            <form
              onSubmit={submitSearch}
              className="relative hidden max-w-xs flex-1 md:block lg:max-w-sm"
              role="search"
            >
              <SearchIcon className="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-[var(--text-muted)]" />
              <input
                type="search"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search candidates…"
                className="ats-input h-9 py-1.5 pl-9 pr-3 shadow-none"
                aria-label="Search candidates"
              />
            </form>

            <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
              <Link
                href="/dashboard/jobs/new"
                className="hidden h-9 items-center rounded-[var(--radius-md)] border border-[var(--border-strong)] bg-[var(--surface)] px-3 text-xs font-semibold text-[var(--text-primary)] transition hover:bg-[var(--surface-muted)] sm:inline-flex"
              >
                New job
              </Link>
              <Link
                href="/dashboard/candidates/new"
                className="inline-flex h-9 items-center rounded-[var(--radius-md)] bg-[var(--accent)] px-3 text-xs font-semibold text-white transition hover:bg-[var(--accent-hover)]"
              >
                Add candidate
              </Link>

              <button
                type="button"
                aria-label={dark ? "Switch to light mode" : "Switch to dark mode"}
                className="inline-flex h-9 w-9 items-center justify-center rounded-[var(--radius-md)] border border-[var(--border)] bg-[var(--surface)] text-[var(--text-secondary)] transition hover:bg-[var(--surface-muted)]"
                onClick={toggleTheme}
              >
                {dark ? <SunIcon className="h-4 w-4" /> : <MoonIcon className="h-4 w-4" />}
              </button>

              <NotificationBell />

              <div className="relative" ref={profileRef}>
                <button
                  type="button"
                  className="inline-flex h-9 items-center gap-2 rounded-[var(--radius-md)] border border-[var(--border)] bg-[var(--surface)] px-1.5 pr-2 transition hover:bg-[var(--surface-muted)]"
                  aria-haspopup="menu"
                  aria-expanded={profileOpen}
                  onClick={() => setProfileOpen((v) => !v)}
                >
                  <Avatar name={displayName} size="sm" />
                  <span className="hidden max-w-[7rem] truncate text-xs font-medium text-[var(--text-secondary)] lg:inline">
                    {displayName}
                  </span>
                </button>
                {profileOpen ? (
                  <div
                    role="menu"
                    className="absolute right-0 z-40 mt-2 w-56 overflow-hidden rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--surface)] py-1 shadow-[var(--shadow-md)]"
                  >
                    <div className="border-b border-[var(--border)] px-3 py-2.5">
                      <p className="truncate text-sm font-semibold text-[var(--text-primary)]">{displayName}</p>
                      {roleLabel ? (
                        <p className="truncate text-xs capitalize text-[var(--text-muted)]">{roleLabel}</p>
                      ) : null}
                    </div>
                    <Link
                      href="/dashboard/settings/ai"
                      role="menuitem"
                      className="block px-3 py-2 text-sm text-[var(--text-secondary)] hover:bg-[var(--surface-muted)]"
                      onClick={() => setProfileOpen(false)}
                    >
                      AI settings
                    </Link>
                    <button
                      type="button"
                      role="menuitem"
                      className="block w-full px-3 py-2 text-left text-sm text-[var(--danger)] hover:bg-[var(--danger-soft)]"
                      onClick={handleLogout}
                    >
                      Log out
                    </button>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        </header>

        <main className={`${container} flex-1 ${pagePadding}`}>
          <div className="ats-page-enter">{children}</div>
        </main>
        <FloatingAssistant />
      </div>
    </div>
  );
}

function OverviewIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M4 13h6V4H4v9zm10 7h6V4h-6v16zM4 20h6v-5H4v5z" />
    </svg>
  );
}

function JobsIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 6V5a2 2 0 012-2h2a2 2 0 012 2v1m-9 0h12a2 2 0 012 2v9a2 2 0 01-2 2H6a2 2 0 01-2-2V8a2 2 0 012-2z" />
    </svg>
  );
}

function CandidatesIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2" />
      <circle cx="9" cy="7" r="3" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M22 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75" />
    </svg>
  );
}

function CalendarIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M8 3v3m8-3v3M4 9h16M6 5h12a2 2 0 012 2v12a2 2 0 01-2 2H6a2 2 0 01-2-2V7a2 2 0 012-2z" />
    </svg>
  );
}

function ReportsIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M4 19V5m4 14V9m4 10v-6m4 6V7m4 12V11" />
    </svg>
  );
}

function SettingsIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <circle cx="12" cy="12" r="3" />
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M19.4 15a1.7 1.7 0 00.3 1.8l.1.1a2 2 0 11-2.8 2.8l-.1-.1a1.7 1.7 0 00-1.8-.3 1.7 1.7 0 00-1 1.5V21a2 2 0 11-4 0v-.1a1.7 1.7 0 00-1-1.5 1.7 1.7 0 00-1.8.3l-.1.1a2 2 0 11-2.8-2.8l.1-.1a1.7 1.7 0 00.3-1.8 1.7 1.7 0 00-1.5-1H3a2 2 0 110-4h.1a1.7 1.7 0 001.5-1 1.7 1.7 0 00-.3-1.8l-.1-.1a2 2 0 112.8-2.8l.1.1a1.7 1.7 0 001.8.3H9a1.7 1.7 0 001-1.5V3a2 2 0 114 0v.1a1.7 1.7 0 001 1.5 1.7 1.7 0 001.8-.3l.1-.1a2 2 0 112.8 2.8l-.1.1a1.7 1.7 0 00-.3 1.8V9c.3.6.9 1 1.5 1H21a2 2 0 110 4h-.1a1.7 1.7 0 00-1.5 1z"
      />
    </svg>
  );
}

function CollapseIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M15 6l-6 6 6 6" />
    </svg>
  );
}

function MenuIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <path strokeLinecap="round" d="M4 7h16M4 12h16M4 17h16" />
    </svg>
  );
}

function SearchIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <circle cx="11" cy="11" r="7" />
      <path strokeLinecap="round" d="M20 20l-3-3" />
    </svg>
  );
}

function SunIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <circle cx="12" cy="12" r="4" />
      <path strokeLinecap="round" d="M12 2v2m0 16v2M4.9 4.9l1.4 1.4m11.4 11.4l1.4 1.4M2 12h2m16 0h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  );
}

function MoonIcon({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M21 14.5A8.5 8.5 0 1111.5 3a7 7 0 009.5 11.5z" />
    </svg>
  );
}
