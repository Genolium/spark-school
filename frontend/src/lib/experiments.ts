"use client";

import { getVisitorId } from "./analytics";

export type ExperimentName = "hero_cta_copy" | "pricing_cta_copy";

export interface ExperimentDefinition {
  name: ExperimentName;
  variants: string[];
}

export const EXPERIMENTS: ExperimentDefinition[] = [
  { name: "hero_cta_copy", variants: ["control", "benefit", "diagnostic"] },
  { name: "pricing_cta_copy", variants: ["control", "action"] },
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

export function getExperimentAssignments(): Record<string, string> {
  if (typeof window === "undefined") return {};
  const result: Record<string, string> = {};
  for (const definition of EXPERIMENTS) {
    result[definition.name] = getExperiment(definition.name);
  }
  return result;
}
