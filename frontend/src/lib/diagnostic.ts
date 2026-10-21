"use client";

import { API_BASE } from "./api-base";
import { getAttributionToken, getVisitorId } from "./analytics";

export interface DiagnosticStep {
  title: string;
  description: string;
}

export interface DiagnosticPlan {
  score: number;
  summary: string;
  warning?: string;
  recommended_tier?: string;
  steps: DiagnosticStep[];
}

export interface DiagnosticResult {
  token: string;
  plan: DiagnosticPlan;
  telegram_url: string;
  plan_url: string;
}

export interface DiagnosticInput {
  age_citizenship: boolean;
  university_year: string;
  english_level: string;
  source?: string;
}

export async function createDiagnostic(input: DiagnosticInput): Promise<DiagnosticResult | null> {
  try {
    const res = await fetch(`${API_BASE}/diagnostic`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ...input,
        visitor_id: getVisitorId(),
        attribution_token: getAttributionToken(),
      }),
    });
    if (!res.ok) return null;
    return (await res.json()) as DiagnosticResult;
  } catch {
    return null;
  }
}

export async function getDiagnosticPlan(token: string): Promise<(DiagnosticResult & { linked: boolean }) | null> {
  try {
    const res = await fetch(`${API_BASE}/diagnostic/${encodeURIComponent(token)}`);
    if (!res.ok) return null;
    return (await res.json()) as DiagnosticResult & { linked: boolean };
  } catch {
    return null;
  }
}
