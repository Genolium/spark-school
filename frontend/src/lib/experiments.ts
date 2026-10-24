"use client";

import { getAttributionToken, getVisitorId } from "./analytics";
import { API_BASE } from "./api-base";
import type { CoursePricing } from "./api-types";
import { ACCELERATOR_LIST_PRICE, ACCELERATOR_LOWER_PRICE } from "./pricing-defaults";

export type ExperimentName = "hero_cta_copy" | "hero_headline" | "pricing_cta_copy" | "pricing_price" | "vip_price";

export interface ExperimentDefinition {
  name: ExperimentName;
  variants: string[];
}

export const EXPERIMENTS: ExperimentDefinition[] = [
  { name: "hero_cta_copy", variants: ["control", "benefit", "diagnostic"] },
  { name: "hero_headline", variants: ["control", "benefit", "urgency"] },
  { name: "pricing_cta_copy", variants: ["control", "action"] },
  { name: "pricing_price", variants: ["control", "lower"] },
  { name: "vip_price", variants: ["control", "mid", "low"] },
];

const STORAGE_PREFIX = "spark_exp_";

function hash(input: string): number {
  let value = 2166136261;
  for (let i = 0; i < input.length; i++) {
    value ^= input.charCodeAt(i);
    value = Math.imul(value, 16777619);
  }
  return value >>> 0;
}

/** Sticky deterministic bucket: the same visitor always sees the same variant. */
export function getExperiment(name: ExperimentName): string {
  const definition = EXPERIMENTS.find((item) => item.name === name);
  if (!definition) return "control";
  if (typeof window === "undefined") return definition.variants[0];

  try {
    const stored = localStorage.getItem(STORAGE_PREFIX + name);
    if (stored && definition.variants.includes(stored)) return stored;

    const visitorId = getVisitorId() || "anonymous";
    const variant = definition.variants[hash(`${visitorId}:${name}`) % definition.variants.length];
    localStorage.setItem(STORAGE_PREFIX + name, variant);
    return variant;
  } catch {
    return definition.variants[0];
  }
}

/**
 * Ask the server for the canonical buckets and persist them locally. The local
 * deterministic fallback is only for the first paint; the server assignment is
 * what analytics and the experiment report use.
 */
export async function syncServerExperiments(): Promise<Record<string, string>> {
  if (typeof window === "undefined") return {};
  const visitorId = getVisitorId();
  if (!visitorId) return {};
  try {
    const res = await fetch(`${API_BASE}/analytics/experiments`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ visitor_id: visitorId, attribution_token: getAttributionToken() }),
    });
    if (!res.ok) return {};
    const json = (await res.json()) as { experiments?: Record<string, string> };
    const assignments = json.experiments || {};
    for (const [name, variant] of Object.entries(assignments)) {
      localStorage.setItem(STORAGE_PREFIX + name, variant);
    }
    return assignments;
  } catch {
    return {};
  }
}

/** Sale price vs list 10 900 ₽. */
export const LOWER_ACCELERATOR_PRICE = ACCELERATOR_LOWER_PRICE;
export const MID_VIP_PRICE = 17990;
export const LOW_VIP_PRICE = 14990;

/** Keep every price surface (cards, modal, checkout fallback) on one variant. */
export function applyPriceExperiment(pricing: CoursePricing): CoursePricing {
  let next = pricing;
  if (getExperiment("pricing_price") === "lower") {
    const list = Math.max(next.accelerator_price || 0, next.accelerator_old_price || 0, ACCELERATOR_LIST_PRICE);
    next = { ...next, accelerator_old_price: list, accelerator_price: LOWER_ACCELERATOR_PRICE };
  }
  const vipVariant = getExperiment("vip_price");
  if (vipVariant === "mid") {
    next = { ...next, vip_price: MID_VIP_PRICE };
  } else if (vipVariant === "low") {
    next = { ...next, vip_price: LOW_VIP_PRICE };
  }
  return next;
}

export function getExperimentAssignments(): Record<string, string> {
  if (typeof window === "undefined") return {};
  const result: Record<string, string> = {};
  for (const definition of EXPERIMENTS) {
    result[definition.name] = getExperiment(definition.name);
  }
  return result;
}
