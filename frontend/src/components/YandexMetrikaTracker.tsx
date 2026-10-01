"use client";

import { usePathname } from "next/navigation";
import { useEffect } from "react";

/**
 * Tracks client-side route changes for Yandex.Metrika in Next.js SPA navigation
 */
export function YandexMetrikaTracker() {
  const pathname = usePathname();

  useEffect(() => {
    if (typeof window !== "undefined" && typeof (window as unknown as { ym?: (...args: unknown[]) => void }).ym === "function") {
      (window as unknown as { ym: (...args: unknown[]) => void }).ym(113185946, "hit", window.location.href);
    }
  }, [pathname]);

  return null;
}
