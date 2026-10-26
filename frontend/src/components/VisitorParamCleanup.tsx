"use client";

import { useEffect } from "react";

import { getVisitorId, stripVisitorParamFromUrl } from "@/lib/analytics";

/**
 * `getVisitorId()` adopts `?spark_vid=` as early as it is needed, but Next's
 * router rewrites its canonical URL from a `useInsertionEffect` while
 * hydrating, which can restore the parameter after a render-phase strip. Run
 * the strip once more from an effect, after that commit, so the address bar
 * stays clean for copied and shared URLs.
 *
 * This effect can run before the page's own effects, so it resolves the id
 * first: the parameter must be adopted before it is removed from the URL.
 */
export function VisitorParamCleanup() {
  useEffect(() => {
    getVisitorId();
    stripVisitorParamFromUrl();
  }, []);

  return null;
}
