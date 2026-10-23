// API base URL and auth token helpers.

export const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// Helper to get admin JWT token from local storage
export function getAdminToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("spark_admin_token");
}

// Helper to get unified auth token (student or admin)
export function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return (
    localStorage.getItem("spark_token") ||
    localStorage.getItem("spark_admin_token") ||
    null
  );
}

// Helper to clear invalid/expired admin auth credentials and trigger re-login
export function handleAdminAuthFailure(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem("spark_admin_token");
  localStorage.removeItem("spark_admin_user");
  sessionStorage.removeItem("spark_admin_unlocked");
  window.dispatchEvent(new CustomEvent("spark:admin-unauthorized"));
}


