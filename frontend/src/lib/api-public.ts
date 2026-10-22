"use client";

import { API_BASE } from "./api-base";
import { getVisitorId } from "./analytics";
import { applyPriceExperiment } from "./experiments";
import type { PlacesStats, CoursePricing, ValidatePromoResponse, UserProfile, AdminStats, PayoutRequest, PromoCode, AffiliateStats, ExperimentMetric, DeadOutboxEvent, FunnelData } from "./api-types";

export const publicApi = {
  async getPlacesStats(): Promise<PlacesStats> {
    try {
      const res = await fetch(`${API_BASE}/stats/places`);
      if (res.ok) {
        return await res.json();
      }
    } catch {
      // Offline fallback
    }
    return {
      total_capacity: 25,
      active_students: 16,
      spots_left: 9,
    };
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

    return applyPriceExperiment({
      basic_price: 3990,
      basic_old_price: 5900,
      accelerator_price: 6900,
      accelerator_old_price: 9900,
      vip_price: 14900,
      vip_old_price: 19900,
      bank_name: "БСПБ (Банк «Санкт-Петербург»)",
    });
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
      let basePrice = 6900;
      if (price && price > 0) {
        basePrice = price;
      } else if (tier) {
        const t = tier.toLowerCase();
        if (t === "vip") basePrice = 14900;
        else if (t === "basic" || t === "базовый") basePrice = 3990;
        else basePrice = 6900;
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
