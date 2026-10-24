"use client";

import { API_BASE } from "./api-base";
import { getVisitorId } from "./analytics";
import { applyPriceExperiment } from "./experiments";
import { DEFAULT_PRICING } from "./pricing-defaults";
import type { PlacesStats, CoursePricing, ValidatePromoResponse, UserProfile, AdminStats, PayoutRequest, PromoCode, AffiliateStats, ExperimentMetric, DeadOutboxEvent, FunnelData } from "./api-types";

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
      const visitorId = getVisitorId();
      const query = visitorId ? `?visitor_id=${encodeURIComponent(visitorId)}` : "";
      const res = await fetch(`${API_BASE}/pricing${query}`);
      if (res.ok) {
        const data = applyPriceExperiment(await res.json());
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
          return applyPriceExperiment(JSON.parse(saved));
        } catch {}
      }
    }

    return applyPriceExperiment({ ...DEFAULT_PRICING });
  },

  async validatePromoCode(code: string, tier?: string, price?: number): Promise<ValidatePromoResponse> {
    try {
      const res = await fetch(`${API_BASE}/promo/validate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code, tier, price }),
      });
      return await res.json();
    } catch {
      // Mock fallback: if code ends with 5 or is START5/IL5/SPARK5, grant 5% discount
      let basePrice = DEFAULT_PRICING.accelerator_price;
      if (price && price > 0) {
        basePrice = price;
      } else if (tier) {
        const t = tier.toLowerCase();
        if (t === "vip") basePrice = DEFAULT_PRICING.vip_price;
        else if (t === "basic" || t === "базовый") basePrice = DEFAULT_PRICING.basic_price || 3990;
        else basePrice = DEFAULT_PRICING.accelerator_price;
      }

      const clean = code.trim().toUpperCase();
      if (clean === "START5" || clean === "IL5" || clean === "SPARK5" || clean.endsWith("5")) {
        const discountAmount = Math.round((basePrice * 5) / 100);
        return {
          valid: true,
          code: clean,
          discount_percent: 5,
          original_price: basePrice,
          discount_amount: discountAmount,
          final_price: basePrice - discountAmount,
          owner_username: "partner",
          message: "Промокод успешно применён!",
        };
      }
      return {
        valid: false,
        code: clean,
        discount_percent: 0,
        original_price: basePrice,
        discount_amount: 0,
        final_price: basePrice,
        message: "Промокод не найден",
      };
    }
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
