"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { isAuthenticated } from "@/lib/auth-storage";

export function useRequireAuth() {
  const router = useRouter();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") return;

    if (!isAuthenticated()) {
      router.replace("/login");
      setReady(false);
      return;
    }

    setReady(true);
  }, [router]);

  return ready;
}
