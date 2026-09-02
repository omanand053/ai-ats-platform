"use client";

import type { ReactNode } from "react";
import { DashboardShell } from "@/components/dashboard/DashboardShell";

/**
 * Persistent dashboard chrome — sidebar/header mount once for all /dashboard/* routes.
 */
export default function DashboardLayout({ children }: { children: ReactNode }) {
  return <DashboardShell>{children}</DashboardShell>;
}
