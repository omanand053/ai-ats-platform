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
import { validateSignup } from "@/lib/validators";
import { signup } from "@/services/auth.service";

function slugify(value: string) {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

export function SignupForm() {
  const router = useRouter();
  const { toasts, show, dismiss } = useToast();
  const [form, setForm] = useState({
    company_name: "",
    company_slug: "",
    first_name: "",
    last_name: "",
    email: "",
    password: "",
  });
  const [slugEdited, setSlugEdited] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (isAuthenticated()) {
      router.replace("/dashboard");
    }
  }, [router]);
  const [loading, setLoading] = useState(false);

  function updateField(field: keyof typeof form, value: string) {
    setForm((prev) => {
      const next = { ...prev, [field]: value };
      if (field === "company_name" && !slugEdited) {
        next.company_slug = slugify(value);
      }
      return next;
    });
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const fieldErrors = validateSignup(form);
    setErrors(fieldErrors);
    if (Object.keys(fieldErrors).length) return;

    setLoading(true);
    try {
      await signup(form);
      show("Account created successfully!");
      router.push("/dashboard");
    } catch (err) {
      const message =
        err instanceof ApiClientError ? err.message : "Signup failed. Please try again.";
      show(message, "error");
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <AuthLayout
        title="Create your account"
        subtitle="Register your company and start screening candidates"
        footer={
          <>
            Already have an account?{" "}
            <Link href="/login" className="font-medium text-blue-600 hover:text-blue-500">
              Log in
            </Link>
          </>
        }
      >
        <form onSubmit={handleSubmit} className="space-y-4" noValidate>
          <Input
            label="Company Name"
            name="company_name"
            value={form.company_name}
            onChange={(e) => updateField("company_name", e.target.value)}
            error={errors.company_name}
            placeholder="Acme Corp"
          />
          <Input
            label="Company Slug"
            name="company_slug"
            value={form.company_slug}
            onChange={(e) => {
              setSlugEdited(true);
              updateField("company_slug", slugify(e.target.value));
            }}
            error={errors.company_slug}
            placeholder="acme-corp"
          />
          <div className="grid gap-4 sm:grid-cols-2">
            <Input
              label="First Name"
              name="first_name"
              value={form.first_name}
              onChange={(e) => updateField("first_name", e.target.value)}
              error={errors.first_name}
              placeholder="Jane"
            />
            <Input
              label="Last Name"
              name="last_name"
              value={form.last_name}
              onChange={(e) => updateField("last_name", e.target.value)}
              error={errors.last_name}
              placeholder="Doe"
            />
          </div>
          <Input
            label="Email"
            name="email"
            type="email"
            autoComplete="email"
            value={form.email}
            onChange={(e) => updateField("email", e.target.value)}
            error={errors.email}
            placeholder="you@company.com"
          />
          <Input
            label="Password"
            name="password"
            type="password"
            autoComplete="new-password"
            value={form.password}
            onChange={(e) => updateField("password", e.target.value)}
            error={errors.password}
            placeholder="Min. 8 characters"
          />
          <Button type="submit" loading={loading}>
            Create account
          </Button>
        </form>
      </AuthLayout>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
