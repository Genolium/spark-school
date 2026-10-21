"use client";

import { API_BASE, getAuthToken } from "./api-base";
import type { PlacesStats, CoursePricing, ValidatePromoResponse, UserProfile, AdminStats, PayoutRequest, PromoCode, AffiliateStats, ExperimentMetric, DeadOutboxEvent, FunnelData } from "./api-types";

export const affiliateApi = {
  async getAffiliateStats(): Promise<AffiliateStats> {
    const token = getAuthToken();
    try {
      const res = await fetch(`${API_BASE}/affiliate/stats`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: "include",
      });
      if (res.ok) return await res.json();
    } catch {
      // Server error
    }

    return {
      referral_code: "",
      referral_link: "",
      current_balance: 0,
      total_earned: 0,
      total_withdrawn: 0,
      level1_referrals: 0,
      level2_referrals: 0,
      total_referrals: 0,
      transactions: [],
      payouts: [],
    };
  },

  async submitAffiliatePayout(amount: number, details: string): Promise<{ success: boolean; message?: string; error?: string; payout?: PayoutRequest }> {
    const token = getAuthToken();
    try {
      const res = await fetch(`${API_BASE}/affiliate/payout`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ amount, details }),
        credentials: "include",
      });

      const data = await res.json();
      if (res.ok) {
        return { success: true, message: data.message, payout: data.payout };
      }
      return { success: false, error: data.error || "Ошибка отправки заявки на вывод" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Сетевая ошибка при отправке заявки на вывод" };
    }
  },
};
