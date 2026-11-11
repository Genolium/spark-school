"use client";

import { API_BASE } from "./api-base";
import { DEFAULT_PRICING } from "./pricing-defaults";
import type { PlacesStats, CoursePricing } from "./api-types";

export const publicApi = {
  async getPlacesStats(): Promise<PlacesStats | null> {
    try {
      const res = await fetch(`${API_BASE}/stats/places`);
      if (res.ok) {
        return await res.json();
      }
    } catch {
      // Show a neutral status rather than invented occupancy.
    }
    return null;
  },

  async getCoursePricing(): Promise<CoursePricing> {
    try {
      const res = await fetch(`${API_BASE}/pricing`);
      if (res.ok) {
        const data: CoursePricing = await res.json();
        if (typeof window !== "undefined") {
          localStorage.setItem("spark_pricing", JSON.stringify(data));
        }
        return data;
      }
    } catch {
      // Offline fallback
    }

    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("spark_pricing");
      if (saved) {
        try {
          return JSON.parse(saved) as CoursePricing;
        } catch {}
      }
    }

    return { ...DEFAULT_PRICING };
  },

  async createYooKassaPayment(data: {
    tier: "basic" | "accelerator" | "vip";
    telegram_username: string;
    telegram_id?: number;
    promo_code?: string;
    email?: string;
    visitor_id?: string;
    attribution_token?: string;
  }): Promise<{ success: boolean; confirmation_url?: string; payment_id?: string; error?: string }> {
    try {
      const res = await fetch(`${API_BASE}/payments/yookassa/create`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      const json = await res.json();
      if (res.ok && json.confirmation_url) {
        return { success: true, confirmation_url: json.confirmation_url, payment_id: json.payment_id };
      }
      return { success: false, error: json.error || "Не удалось создать платёж в ЮKassa" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Ошибка подключения к платёжному шлюзу" };
    }
  },
};
