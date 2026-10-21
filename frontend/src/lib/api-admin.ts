"use client";

import { API_BASE, getAdminToken } from "./api-base";
import type { PlacesStats, CoursePricing, ValidatePromoResponse, UserProfile, AdminStats, PayoutRequest, PromoCode, AffiliateStats, ExperimentMetric, DeadOutboxEvent, FunnelData } from "./api-types";

export const adminApi = {
  async updateCoursePricing(pricing: CoursePricing): Promise<{ success: boolean; pricing?: CoursePricing; error?: string }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/pricing`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(pricing),
        credentials: "include",
      });
      const data = await res.json();
      if (res.ok) {
        if (typeof window !== "undefined") {
          localStorage.setItem("spark_pricing", JSON.stringify(data));
        }
        return { success: true, pricing: data };
      }
      return { success: false, error: data.error || "Не удалось обновить цены" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Сетевая ошибка при обновлении цен" };
    }
  },

  async adminLogin(username: string, password: string, totpCode?: string): Promise<{ success: boolean; token?: string; user?: UserProfile; error?: string; retryAfter?: number; totpRequired?: boolean }> {
    try {
      const res = await fetch(`${API_BASE}/admin/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password, totp_code: totpCode || "" }),
        credentials: "include",
      });

      if (res.ok) {
        const data = await res.json();
        if (typeof window !== "undefined") {
          localStorage.setItem("spark_admin_token", data.token);
          localStorage.setItem("spark_admin_user", JSON.stringify(data.user));
          sessionStorage.setItem("spark_admin_unlocked", "true");
        }
        return { success: true, token: data.token, user: data.user };
      }

      if (res.status === 429) {
        const err = await res.json().catch(() => ({ error: "Слишком много попыток входа. Пожалуйста, подождите 60 секунд." }));
        return {
          success: false,
          error: err.error || "Слишком много попыток входа (лимит: 5 в минуту). Пожалуйста, подождите.",
          retryAfter: err.retry_after_seconds || 60,
        };
      }

      const err = await res.json().catch(() => ({ error: "Ошибка авторизации" }));
      return { success: false, error: err.error || "Неверный логин или пароль" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Ошибка подключения к серверу авторизации" };
    }
  },

  adminLogout() {
    if (typeof window !== "undefined") {
      const token = getAdminToken();
      // Best-effort server-side revocation; local state is cleared regardless.
      void fetch(`${API_BASE}/admin/logout`, {
        method: "POST",
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: "include",
      }).catch(() => {});
      localStorage.removeItem("spark_admin_token");
      localStorage.removeItem("spark_admin_user");
      sessionStorage.removeItem("spark_admin_unlocked");
    }
  },

  async getAdminStats(): Promise<AdminStats> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/stats`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: "include",
      });
      if (res.ok) {
        const data: AdminStats = await res.json();
        return {
          ...data,
          daily_stats: data.daily_stats || [],
          today_revenue: data.today_revenue ?? 0,
          today_purchases: data.today_purchases ?? 0,
          week_revenue: data.week_revenue ?? 0,
          month_revenue: data.month_revenue ?? 0,
          average_order_value: data.average_order_value ?? 0,
          promo_usage_count: data.promo_usage_count ?? 0,
          promo_saved_amount: data.promo_saved_amount ?? 0,
          tier_breakdown: data.tier_breakdown ?? {
            basic: { count: 0, revenue: 0, percentage: 0 },
            accelerator: { count: 0, revenue: 0, percentage: 0 },
            vip: { count: 0, revenue: 0, percentage: 0 },
          },
        };
      }
    } catch {
      // Server error or network offline
    }

    return {
      gross_volume: 0,
      active_students: 0,
      total_registered: 0,
      conversion_rate: 0,
      pending_payouts_count: 0,
      pending_payouts_sum: 0,
      today_revenue: 0,
      today_purchases: 0,
      week_revenue: 0,
      month_revenue: 0,
      average_order_value: 0,
      promo_usage_count: 0,
      promo_saved_amount: 0,
      tier_breakdown: {
        basic: { count: 0, revenue: 0, percentage: 0 },
        accelerator: { count: 0, revenue: 0, percentage: 0 },
        vip: { count: 0, revenue: 0, percentage: 0 },
      },
      daily_stats: [],
    };
  },

  // Admin students list with search and filters

  async getAdminStudents(params?: {
    q?: string;
    tier?: string;
    access?: string;
    receipt_status?: string;
    receipt_issued?: string;
    sort_by?: string;
    sort_dir?: "asc" | "desc";
    page?: number;
    pageSize?: number;
  } | string): Promise<UserProfile[]> {
    const queryObj = typeof params === "string" ? { q: params } : (params || {});
    const {
      q = "",
      tier = "",
      access = "",
      receipt_status = "",
      receipt_issued = "",
      sort_by = "purchase_date",
      sort_dir = "desc",
      page,
      pageSize,
    } = queryObj;

    const token = getAdminToken();
    try {
      const sp = new URLSearchParams();
      if (q) sp.set("q", q);
      if (tier && tier !== "all") sp.set("tier", tier);
      if (access && access !== "all") sp.set("access", access);
      if (receipt_status && receipt_status !== "all") sp.set("receipt_status", receipt_status);
      if (receipt_issued && receipt_issued !== "all") sp.set("receipt_issued", receipt_issued);
      if (sort_by) sp.set("sort_by", sort_by);
      if (sort_dir) sp.set("sort_dir", sort_dir);
      if (page) sp.set("page", String(page));
      if (pageSize) sp.set("pageSize", String(pageSize));

      const res = await fetch(`${API_BASE}/admin/users?${sp.toString()}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: "include",
      });
      if (res.ok) {
        const data = await res.json();
        return Array.isArray(data.users) ? data.users : [];
      }
    } catch {
      // Fallback
    }

    return [];
  },

  async toggleReceiptIssued(
    userId: number,
    issued?: boolean
  ): Promise<{ success: boolean; receipt_issued: boolean; receipt_issued_at?: string; user?: UserProfile; error?: string }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/users/${userId}/receipt-issued`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(issued !== undefined ? { receipt_issued: issued } : {}),
        credentials: "include",
      });
      if (res.ok) {
        return await res.json();
      }
      const data = await res.json().catch(() => ({}));
      return { success: false, receipt_issued: false, error: data.error || "Ошибка сохранения статуса чека" };
    } catch (e: any) {
      return { success: false, receipt_issued: false, error: e?.message || "Сетевая ошибка сохранения статуса чека" };
    }
  },

  async issueReceipt(
    userId: number,
    payload: {
      receipt_url?: string;
      receipt_number?: string;
      amount?: number;
      service_name?: string;
      send_telegram?: boolean;
    }
  ): Promise<{
    success: boolean;
    message?: string;
    telegram_sent?: boolean;
    receipt_issued?: boolean;
    receipt_issued_at?: string;
    receipt_url?: string;
    receipt_number?: string;
    user?: UserProfile;
    error?: string;
  }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/users/${userId}/issue-receipt`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(payload),
        credentials: "include",
      });
      if (res.ok) {
        return await res.json();
      }
      const data = await res.json().catch(() => ({}));
      return { success: false, error: data.error || "Ошибка выписки чека" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Сетевая ошибка выписки чека" };
    }
  },

  async toggleUserAccess(
    userId: number,
    hasAccess: boolean,
    extra?: { tier?: string; paid_amount?: number; promo_code?: string }
  ): Promise<boolean> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/users/${userId}/access`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ has_access: hasAccess, ...extra }),
        credentials: "include",
      });
      if (res.ok) return true;
      return false;
    } catch {
      return false;
    }
  },

  async getAdminPayouts(): Promise<PayoutRequest[]> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/payouts`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: "include",
      });
      if (res.ok) {
        const data = await res.json();
        return Array.isArray(data.payouts) ? data.payouts : [];
      }
    } catch {
      // Mock payouts removed
    }

    return [];
  },

  async approvePayout(id: number): Promise<boolean> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/payouts/${id}/approve`, {
        method: "POST",
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: "include",
      });
      if (res.ok) return true;
      return false;
    } catch {
      return false;
    }
  },

  async rejectPayout(id: number): Promise<boolean> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/payouts/${id}/reject`, {
        method: "POST",
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: "include",
      });
      if (res.ok) return true;
      return false;
    } catch {
      return false;
    }
  },

  async triggerAdminDigest(): Promise<{ success: boolean; message?: string; error?: string }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/digest`, {
        method: "POST",
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: "include",
      });
      if (res.ok) {
        const data = await res.json();
        return { success: true, message: data.message || "Дайджест отправлен" };
      }
      const data = await res.json().catch(() => ({}));
      return { success: false, error: data.error || "Ошибка отправки дайджеста" };
    } catch (e: any) {
      return { success: false, error: e.message || "Сетевая ошибка" };
    }
  },

  async getAdminPromoCodes(): Promise<PromoCode[]> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/promos`, {
        headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        credentials: "include",
      });
      if (res.ok) return await res.json();
      return [];
    } catch {
      return [];
    }
  },

  async createAdminPromoCode(data: { code: string; owner_username?: string; discount_percent?: number; reward_amount?: number }): Promise<{ success: boolean; promo?: PromoCode; error?: string }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/promos`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(data),
        credentials: "include",
      });
      const json = await res.json();
      if (res.ok) return { success: true, promo: json };
      return { success: false, error: json.error || "Ошибка создания промокода" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Сетевая ошибка создания промокода" };
    }
  },

  async createAdminUser(data: {
    telegram_id: number;
    first_name: string;
    last_name?: string;
    username?: string;
    role?: "student" | "admin";
    has_access?: boolean;
    tier?: "basic" | "accelerator" | "vip";
    paid_amount?: number;
    promo_code?: string;
    receipt_issued?: boolean;
  }): Promise<{ success: boolean; user?: UserProfile; error?: string }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/users`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(data),
        credentials: "include",
      });
      const json = await res.json();
      if (res.ok) return { success: true, user: json };
      return { success: false, error: json.error || "Не удалось создать пользователя" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Сетевая ошибка создания пользователя" };
    }
  },

  async updateAdminUser(
    id: number,
    data: {
      first_name?: string;
      last_name?: string;
      username?: string;
      role?: "student" | "admin";
      has_access?: boolean;
      tier?: "basic" | "accelerator" | "vip";
      paid_amount?: number;
      promo_code?: string;
      receipt_issued?: boolean;
      receipt_url?: string;
      receipt_number?: string;
    }
  ): Promise<{ success: boolean; user?: UserProfile; error?: string }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/users/${id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(data),
        credentials: "include",
      });
      const json = await res.json();
      if (res.ok) return { success: true, user: json };
      return { success: false, error: json.error || "Не удалось обновить данные" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Сетевая ошибка обновления данных" };
    }
  },

  async deleteAdminUser(id: number): Promise<{ success: boolean; error?: string }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/users/${id}`, {
        method: "DELETE",
        headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        credentials: "include",
      });
      if (res.ok) return { success: true };
      const json = await res.json();
      return { success: false, error: json.error || "Не удалось удалить пользователя" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Сетевая ошибка удаления пользователя" };
    }
  },

  async updateAdminPromoCode(
    id: number,
    data: {
      code?: string;
      owner_username?: string;
      discount_percent?: number;
      reward_amount?: number;
      is_active?: boolean;
    }
  ): Promise<{ success: boolean; promo?: PromoCode; error?: string }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/promos/${id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(data),
        credentials: "include",
      });
      const json = await res.json();
      if (res.ok) return { success: true, promo: json };
      return { success: false, error: json.error || "Не удалось сохранить промокод" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Сетевая ошибка сохранения промокода" };
    }
  },

  async deleteAdminPromoCode(id: number): Promise<{ success: boolean; error?: string }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/promos/${id}`, {
        method: "DELETE",
        headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        credentials: "include",
      });
      if (res.ok) return { success: true };
      const json = await res.json();
      return { success: false, error: json.error || "Не удалось удалить промокод" };
    } catch (e: any) {
      return { success: false, error: e?.message || "Сетевая ошибка удаления промокода" };
    }
  },

  async toggleAdminPromoCode(id: number): Promise<{ success: boolean; promo?: PromoCode }> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/promos/${id}/toggle`, {
        method: "POST",
        headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        credentials: "include",
      });
      if (res.ok) {
        const promo = await res.json();
        return { success: true, promo };
      }
      return { success: false };
    } catch {
      return { success: false };
    }
  },

  async getFunnel(filters: {
    from?: string;
    to?: string;
    source?: string;
    campaign?: string;
    tier?: string;
    promo?: string;
  } = {}): Promise<FunnelData | null> {
    const token = getAdminToken();
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters)) {
      if (value) params.set(key, value);
    }
    try {
      const res = await fetch(`${API_BASE}/admin/analytics/funnel?${params.toString()}`, {
        headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        credentials: "include",
      });
      if (!res.ok) return null;
      return (await res.json()) as FunnelData;
    } catch {
      return null;
    }
  },

  async getExperiments(): Promise<ExperimentMetric[] | null> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/analytics/experiments`, {
        headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        credentials: "include",
      });
      if (!res.ok) return null;
      const json = await res.json();
      return (json.experiments || []) as ExperimentMetric[];
    } catch {
      return null;
    }
  },

  async getDeadOutbox(): Promise<DeadOutboxEvent[] | null> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/outbox/dead`, {
        headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        credentials: "include",
      });
      if (!res.ok) return null;
      const json = await res.json();
      return (json.events || []) as DeadOutboxEvent[];
    } catch {
      return null;
    }
  },

  async retryOutbox(id: number): Promise<boolean> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/outbox/${id}/retry`, {
        method: "POST",
        headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        credentials: "include",
      });
      return res.ok;
    } catch {
      return false;
    }
  },
};
