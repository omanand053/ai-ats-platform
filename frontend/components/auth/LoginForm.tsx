"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { AuthLayout } from "@/components/auth/AuthLayout";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { ToastContainer } from "@/components/ui/Toast";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import { isAuthenticated } from "@/lib/auth-storage";
import { validateLogin } from "@/lib/validators";
import { login } from "@/services/auth.service";

export function LoginForm() {
  const router = useRouter();
  const { toasts, show, dismiss } = useToast();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isAuthenticated()) {
      router.replace("/dashboard");
    }
  }, [router]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const fieldErrors = validateLogin(email, password);
    setErrors(fieldErrors);
    if (Object.keys(fieldErrors).length) return;

    setLoading(true);
    try {
      await login({ email, password });
      show("Logged in successfully!");
      router.push("/dashboard");
    } catch (err) {
      const message =
        err instanceof ApiClientError ? err.message : "Login failed. Please try again.";
      show(message, "error");
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <AuthLayout
        title="Welcome back"
        subtitle="Sign in to manage your hiring pipeline"
        footer={
          <>
            Don&apos;t have an account?{" "}
            <Link href="/signup" className="font-medium text-blue-600 hover:text-blue-500">
              Sign up
            </Link>
          </>
        }
      >
        <form onSubmit={handleSubmit} className="space-y-5" noValidate>
          <Input
            label="Email"
            name="email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            error={errors.email}
            placeholder="you@company.com"
          />
          <Input
            label="Password"
            name="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={errors.password}
            placeholder="••••••••"
          />
          <Button type="submit" loading={loading}>
            Login
          </Button>
        </form>
      </AuthLayout>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
