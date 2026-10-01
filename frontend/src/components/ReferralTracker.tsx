"use client";

import React, { Suspense, useEffect } from "react";
import { useSearchParams } from "next/navigation";

function ReferralTrackerContent() {
  const searchParams = useSearchParams();

  useEffect(() => {
    const ref = searchParams?.get("ref");
    if (ref) {
      try {
        localStorage.setItem("spark_ref", ref);
        document.cookie = `spark_ref=${encodeURIComponent(ref)}; max-age=${30 * 24 * 60 * 60}; path=/; SameSite=Lax`;
      } catch (e) {
        // Local storage / cookie access handling
      }
    }
  }, [searchParams]);

  return null;
}

export function ReferralTracker() {
  return (
    <Suspense fallback={null}>
      <ReferralTrackerContent />
    </Suspense>
  );
}
