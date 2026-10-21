// Client API service with transparent local mock fallback for offline and preview testing

export interface UserProfile {
  id: number;
  telegram_id: number;
  username: string;
  first_name: string;
  last_name?: string;
  photo_url?: string;
  role: "student" | "admin";
  has_access: boolean;
  access_granted_at?: string;
  tier?: "basic" | "accelerator" | "vip";
  paid_amount?: number;
  promo_code?: string;
  receipt_status?: "pending" | "approved" | "rejected" | "none" | string;
  created_at?: string;
  receipt_issued?: boolean;
  receipt_issued_at?: string;
  receipt_url?: string;
  receipt_number?: string;
  has_audit_upsell?: boolean;
}

export interface DailyRevenueStat {
  date: string;
  date_label: string;
  revenue: number;
  purchases: number;
  cumulative_revenue: number;
  accelerator_count?: number;
  vip_count?: number;
}

export interface TierBreakdownItem {
  count: number;
  revenue: number;
  percentage: number;
}

export interface AdminStats {
  gross_volume: number;
  active_students: number;
  total_registered: number;
  conversion_rate: number;
  pending_payouts_count: number;
  pending_payouts_sum: number;
  today_revenue?: number;
  today_purchases?: number;
  week_revenue?: number;
  month_revenue?: number;
  average_order_value?: number;
  promo_usage_count?: number;
  promo_saved_amount?: number;
  tier_breakdown?: {
    basic?: TierBreakdownItem;
    accelerator: TierBreakdownItem;
    vip: TierBreakdownItem;
  };
  daily_stats?: DailyRevenueStat[];
}

export interface PayoutRequest {
  id: number;
  user_id: number;
  user?: UserProfile;
  amount: number;
  payment_details: string;
  status: "pending" | "paid" | "rejected";
  created_at: string;
}

export interface AffiliateTransaction {
  id: number;
  buyer_id: number;
  level: number;
  amount: number;
  created_at: string;
}

export interface AffiliateStats {
  referral_code: string;
  referral_link: string;
  current_balance: number;
  total_earned: number;
  total_withdrawn: number;
  level1_referrals: number;
  level2_referrals: number;
  total_referrals: number;
  transactions?: AffiliateTransaction[];
  payouts?: PayoutRequest[];
}

export interface PromoCode {
  id: number;
  code: string;
  discount_percent: number;
  owner_telegram_id?: number;
  owner_username?: string;
  uses_count: number;
  reward_amount: number;
  is_active: boolean;
  created_at: string;
  total_revenue?: number;
  students?: string[];
}

export interface ValidatePromoResponse {
  valid: boolean;
  code: string;
  discount_percent: number;
  original_price: number;
  discount_amount: number;
  final_price: number;
  owner_username?: string;
  message?: string;
}

export type PromoValidationResult = ValidatePromoResponse;

export interface PlacesStats {
  total_capacity: number;
  active_students: number;
  spots_left: number;
}

export interface CoursePricing {
  basic_price?: number;
  basic_old_price?: number;
  accelerator_price: number;
  accelerator_old_price: number;
  vip_price: number;
  vip_old_price: number;
  bank_name?: string;
  updated_at?: string;
}

export function generateMockDailyStats(targetGross: number = 289800): DailyRevenueStat[] {
  const stats: DailyRevenueStat[] = [];
  const now = new Date();
  const ruMonths = ["янв", "фев", "мар", "апр", "май", "июн", "июл", "авг", "сен", "окт", "ноя", "дек"];

  const pattern = [
    { dayOffset: 29, rev: 0, p: 0, acc: 0, vip: 0 },
    { dayOffset: 28, rev: 6555, p: 1, acc: 1, vip: 0 },
    { dayOffset: 27, rev: 0, p: 0, acc: 0, vip: 0 },
    { dayOffset: 26, rev: 14155, p: 1, acc: 0, vip: 1 },
    { dayOffset: 25, rev: 6900, p: 1, acc: 1, vip: 0 },
    { dayOffset: 24, rev: 13110, p: 2, acc: 2, vip: 0 },
    { dayOffset: 23, rev: 0, p: 0, acc: 0, vip: 0 },
    { dayOffset: 22, rev: 6555, p: 1, acc: 1, vip: 0 },
    { dayOffset: 21, rev: 21055, p: 2, acc: 1, vip: 1 },
    { dayOffset: 20, rev: 6900, p: 1, acc: 1, vip: 0 },
    { dayOffset: 19, rev: 0, p: 0, acc: 0, vip: 0 },
    { dayOffset: 18, rev: 13455, p: 2, acc: 2, vip: 0 },
    { dayOffset: 17, rev: 6555, p: 1, acc: 1, vip: 0 },
    { dayOffset: 16, rev: 14900, p: 1, acc: 0, vip: 1 },
    { dayOffset: 15, rev: 6900, p: 1, acc: 1, vip: 0 },
    { dayOffset: 14, rev: 13110, p: 2, acc: 2, vip: 0 },
    { dayOffset: 13, rev: 0, p: 0, acc: 0, vip: 0 },
    { dayOffset: 12, rev: 6555, p: 1, acc: 1, vip: 0 },
    { dayOffset: 11, rev: 14155, p: 1, acc: 0, vip: 1 },
    { dayOffset: 10, rev: 13455, p: 2, acc: 2, vip: 0 },
    { dayOffset: 9, rev: 6900, p: 1, acc: 1, vip: 0 },
    { dayOffset: 8, rev: 0, p: 0, acc: 0, vip: 0 },
    { dayOffset: 7, rev: 13110, p: 2, acc: 2, vip: 0 },
    { dayOffset: 6, rev: 6555, p: 1, acc: 1, vip: 0 },
    { dayOffset: 5, rev: 21055, p: 2, acc: 1, vip: 1 },
    { dayOffset: 4, rev: 6900, p: 1, acc: 1, vip: 0 },
    { dayOffset: 3, rev: 13455, p: 2, acc: 2, vip: 0 },
    { dayOffset: 2, rev: 14155, p: 1, acc: 0, vip: 1 },
    { dayOffset: 1, rev: 13110, p: 2, acc: 2, vip: 0 },
    { dayOffset: 0, rev: 13455, p: 2, acc: 2, vip: 0 },
  ];

  // Scale revenues proportionally to targetGross
  const rawSum = pattern.reduce((s, x) => s + x.rev, 0);
  const factor = rawSum > 0 ? targetGross / rawSum : 1;

  let cumulative = 0;
  for (const item of pattern) {
    const d = new Date(now);
    d.setDate(d.getDate() - item.dayOffset);
    const dateStr = d.toISOString().split("T")[0];
    const label = `${String(d.getDate()).padStart(2, "0")} ${ruMonths[d.getMonth()]}`;
    const scaledRev = item.rev > 0 ? Math.round(item.rev * factor) : 0;
    cumulative += scaledRev;
    stats.push({
      date: dateStr,
      date_label: label,
      revenue: scaledRev,
      purchases: item.p,
      cumulative_revenue: cumulative,
      accelerator_count: item.acc,
      vip_count: item.vip,
    });
  }

  return stats;
}

export const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// Helper to get admin JWT token from local storage
export function getAdminToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("spark_admin_token");
}

// Helper to get unified auth token (student or admin)
export function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return (
    localStorage.getItem("spark_token") ||
    localStorage.getItem("spark_admin_token") ||
    null
  );
}

export interface FunnelStepMetric {
  step: string;
  count: number;
  conversion_from_previous: number;
}

export interface FunnelData {
  filters: { from: string; to: string; source: string; campaign: string; tier: string; promo: string };
  steps: FunnelStepMetric[];
  payments: { count: number; revenue: number; average: number; source: string };
}

export const api = {
  // Public places counter from backend database
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

  // Public & Admin Course Pricing
  async getCoursePricing(): Promise<CoursePricing> {
    try {
      const res = await fetch(`${API_BASE}/pricing`);
      if (res.ok) {
        const data = await res.json();
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
          return JSON.parse(saved);
        } catch {}
      }
    }

    return {
      basic_price: 3990,
      basic_old_price: 5900,
      accelerator_price: 6900,
      accelerator_old_price: 9900,
      vip_price: 14900,
      vip_old_price: 19900,
      bank_name: "БСПБ (Банк «Санкт-Петербург»)",
    };
  },

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

  // Admin Login with Username & Password and Rate Limit handling
  async adminLogin(username: string, password: string): Promise<{ success: boolean; token?: string; user?: UserProfile; error?: string; retryAfter?: number }> {
    try {
      const res = await fetch(`${API_BASE}/admin/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
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

  // Admin Logout
  adminLogout() {
    if (typeof window !== "undefined") {
      localStorage.removeItem("spark_admin_token");
      localStorage.removeItem("spark_admin_user");
      sessionStorage.removeItem("spark_admin_unlocked");
    }
  },

  // Admin Stats
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
  // Admin students list with search, filters, and dynamic sorting
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

  // Toggle or set self-employed fiscal receipt issued status
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

  // Issue and send official fiscal receipt to student via Telegram bot
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

  // Toggle user access in Admin
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

  // Admin payouts list
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

  // Admin payout approve
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

  // Admin payout reject
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

  // Admin trigger daily curator digest
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

  // Affiliate Stats
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

  // Affiliate Submit Payout
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

  // Promo Codes API
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

  async createYooKassaPayment(data: {
    tier: "basic" | "accelerator" | "vip";
    telegram_username: string;
    telegram_id?: number;
    promo_code?: string;
    email?: string;
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
};
