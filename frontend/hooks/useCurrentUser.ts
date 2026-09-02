"use client";

import { useCallback, useEffect, useState } from "react";
import { getStoredUser, isAuthenticated, setStoredUser } from "@/lib/auth-storage";
import type { User } from "@/lib/types";
import { getMe } from "@/services/auth.service";

export function useCurrentUser() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    if (typeof window === "undefined") {
      setLoading(false);
      return null;
    }

    if (!isAuthenticated()) {
      setUser(null);
      setLoading(false);
      return null;
    }

    const cached = getStoredUser();
    if (cached) {
      setUser(cached as User);
    }

    try {
      const me = await getMe();
      setStoredUser({
        id: me.id,
        email: me.email,
        role: me.role,
        first_name: me.first_name,
        last_name: me.last_name,
      });
      setUser(me);
      return me;
    } catch {
      return cached as User | null;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const role = user?.role?.toLowerCase() ?? "";

  return {
    user,
    loading,
    refresh,
    role,
    isAdmin: role === "admin",
  };
}
