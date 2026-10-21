"use client";

import { useEffect } from "react";
import { reportClientError } from "@/lib/errorTracking";

export function ErrorReporter() {
  useEffect(() => {
    const onError = (event: ErrorEvent) => {
      reportClientError(event.message || "unknown window error", event.error?.stack);
    };
    const onRejection = (event: PromiseRejectionEvent) => {
      const reason = event.reason;
      reportClientError(
        typeof reason === "string" ? reason : reason?.message || "unhandled promise rejection",
        reason?.stack,
      );
    };
    window.addEventListener("error", onError);
    window.addEventListener("unhandledrejection", onRejection);
    return () => {
      window.removeEventListener("error", onError);
      window.removeEventListener("unhandledrejection", onRejection);
    };
  }, []);
  return null;
}
