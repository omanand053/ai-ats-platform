export interface FieldErrors {
  [key: string]: string;
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const SLUG_RE = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

export function validateLogin(email: string, password: string): FieldErrors {
  const errors: FieldErrors = {};
  if (!email.trim()) errors.email = "Email is required";
  else if (!EMAIL_RE.test(email)) errors.email = "Enter a valid email";
  if (!password) errors.password = "Password is required";
  return errors;
}

export function validateSignup(data: {
  company_name: string;
  company_slug: string;
  first_name: string;
  last_name: string;
  email: string;
  password: string;
}): FieldErrors {
  const errors: FieldErrors = {};
  if (!data.company_name.trim()) errors.company_name = "Company name is required";
  if (!data.company_slug.trim()) errors.company_slug = "Company slug is required";
  else if (!SLUG_RE.test(data.company_slug))
    errors.company_slug = "Use lowercase letters, numbers, and hyphens only";
  if (!data.first_name.trim()) errors.first_name = "First name is required";
  if (!data.last_name.trim()) errors.last_name = "Last name is required";
  if (!data.email.trim()) errors.email = "Email is required";
  else if (!EMAIL_RE.test(data.email)) errors.email = "Enter a valid email";
  if (!data.password) errors.password = "Password is required";
  else if (data.password.length < 8)
    errors.password = "Password must be at least 8 characters";
  return errors;
}
