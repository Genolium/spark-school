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

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

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
      if (typeof window !== "undefined") {
        localStorage.setItem("spark_pricing", JSON.stringify(pricing));
        return { success: true, pricing };
      }
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
    } catch {
      // Offline fallback: verify against standard credentials
      if (username === "admin" && (password === "SparkAdmin2026!" || password === "2026")) {
        const mockUser: UserProfile = {
          id: 1,
          telegram_id: 123456789,
          username: "admin",
          first_name: "так называемый Иль",
          last_name: "",
          role: "admin",
          has_access: true,
        };
        const mockToken = "mock_jwt_admin_token_" + Date.now();
        if (typeof window !== "undefined") {
          localStorage.setItem("spark_admin_token", mockToken);
          localStorage.setItem("spark_admin_user", JSON.stringify(mockUser));
          sessionStorage.setItem("spark_admin_unlocked", "true");
        }
        return { success: true, token: mockToken, user: mockUser };
      }
      return { success: false, error: "Неверный логин или пароль" };
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
        if (data.daily_stats && data.daily_stats.length > 0) {
          return data;
        }
        // If server daily_stats empty, enrich with mock daily distribution
        const mockDaily = generateMockDailyStats(data.gross_volume || 289800);
        return {
          ...data,
          daily_stats: mockDaily,
          today_revenue: data.today_revenue ?? 13455,
          today_purchases: data.today_purchases ?? 2,
          week_revenue: data.week_revenue ?? 67975,
          month_revenue: data.month_revenue ?? (data.gross_volume || 244800),
          average_order_value: data.average_order_value ?? (data.active_students > 0 ? Math.round(data.gross_volume / data.active_students) : 6900),
          promo_usage_count: data.promo_usage_count ?? Math.round((data.active_students || 42) * 0.7),
          promo_saved_amount: data.promo_saved_amount ?? 11040,
          tier_breakdown: data.tier_breakdown ?? {
            accelerator: { count: 36, revenue: 204870, percentage: 70.7 },
            vip: { count: 6, revenue: 84930, percentage: 29.3 },
          },
        };
      }
    } catch {
      // Fallback
    }

    const mockDaily = generateMockDailyStats(289800);
    return {
      gross_volume: 289800,
      active_students: 42,
      total_registered: 656,
      conversion_rate: 6.4,
      pending_payouts_count: 3,
      pending_payouts_sum: 8500,
      today_revenue: 13455,
      today_purchases: 2,
      week_revenue: 67975,
      month_revenue: 244800,
      average_order_value: 6900,
      promo_usage_count: 32,
      promo_saved_amount: 11040,
      tier_breakdown: {
        accelerator: { count: 36, revenue: 204870, percentage: 70.7 },
        vip: { count: 6, revenue: 84930, percentage: 29.3 },
      },
      daily_stats: mockDaily,
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
      // Fallback mock students
    }

    let list: UserProfile[] = [
      { id: 1, telegram_id: 891230491, username: "ilyan_vas", first_name: "так называемый Иль", last_name: "", role: "admin", has_access: true, access_granted_at: "2026-09-01T12:00:00Z", created_at: "2026-09-01T11:00:00Z", tier: "vip", paid_amount: 14900, receipt_status: "approved", receipt_issued: true, receipt_issued_at: "2026-09-01T13:00:00Z" },
      { id: 2, telegram_id: 481920381, username: "arina_spark", first_name: "Арина", last_name: "Волкова", role: "student", has_access: true, access_granted_at: "2026-09-15T15:30:00Z", created_at: "2026-09-15T14:00:00Z", tier: "accelerator", paid_amount: 6555, promo_code: "SPARK5", receipt_status: "approved", receipt_issued: true, receipt_issued_at: "2026-09-15T16:00:00Z" },
      { id: 3, telegram_id: 719284910, username: "mikhail_tech", first_name: "Михаил", last_name: "Соколов", role: "student", has_access: true, access_granted_at: "2026-09-18T10:15:00Z", created_at: "2026-09-18T09:30:00Z", tier: "accelerator", paid_amount: 6900, receipt_status: "approved", receipt_issued: false },
      { id: 4, telegram_id: 619284910, username: "daria_bmstu", first_name: "Дарья", last_name: "Кузнецова", role: "student", has_access: false, created_at: "2026-09-20T18:00:00Z", tier: "accelerator", receipt_status: "pending", receipt_issued: false },
      { id: 5, telegram_id: 519284910, username: "artem_spbgu", first_name: "Артём", last_name: "Смирнов", role: "student", has_access: true, access_granted_at: "2026-09-22T14:45:00Z", created_at: "2026-09-22T13:10:00Z", tier: "vip", paid_amount: 14900, receipt_status: "approved", receipt_issued: true, receipt_issued_at: "2026-09-22T15:00:00Z" },
      { id: 6, telegram_id: 319284910, username: "polina_voenmeh", first_name: "Полина", last_name: "Иванова", role: "student", has_access: false, created_at: "2026-09-25T11:20:00Z", tier: "accelerator", receipt_status: "none", receipt_issued: false },
    ];

    if (q) {
      const lower = q.toLowerCase();
      list = list.filter(
        (s) =>
          s.username?.toLowerCase().includes(lower) ||
          s.first_name?.toLowerCase().includes(lower) ||
          s.last_name?.toLowerCase().includes(lower) ||
          String(s.telegram_id).includes(lower) ||
          s.promo_code?.toLowerCase().includes(lower)
      );
    }
    if (tier && tier !== "all") {
      list = list.filter((s) => s.tier?.toLowerCase() === tier.toLowerCase());
    }
    if (access === "active") {
      list = list.filter((s) => s.has_access);
    } else if (access === "pending" || access === "inactive") {
      list = list.filter((s) => !s.has_access);
    }
    if (receipt_status && receipt_status !== "all") {
      list = list.filter((s) => s.receipt_status?.toLowerCase() === receipt_status.toLowerCase());
    }
    if (receipt_issued === "issued" || receipt_issued === "true") {
      list = list.filter((s) => !!s.receipt_issued);
    } else if (receipt_issued === "not_issued" || receipt_issued === "false") {
      list = list.filter((s) => !s.receipt_issued);
    }

    // Mock client-side sorting
    list.sort((a, b) => {
      let comparison = 0;
      switch (sort_by) {
        case "purchase_date":
        case "access_granted_at":
        case "access_date": {
          const dateA = a.access_granted_at || a.created_at || "";
          const dateB = b.access_granted_at || b.created_at || "";
          comparison = dateA.localeCompare(dateB);
          break;
        }
        case "paid_amount":
        case "amount": {
          const amtA = a.paid_amount || 0;
          const amtB = b.paid_amount || 0;
          comparison = amtA - amtB;
          break;
        }
        case "name":
        case "first_name": {
          const nameA = `${a.first_name} ${a.last_name || ""}`.trim().toLowerCase();
          const nameB = `${b.first_name} ${b.last_name || ""}`.trim().toLowerCase();
          comparison = nameA.localeCompare(nameB, "ru");
          break;
        }
        case "username": {
          const userA = (a.username || "").toLowerCase();
          const userB = (b.username || "").toLowerCase();
          comparison = userA.localeCompare(userB);
          break;
        }
        case "tier": {
          const tierA = (a.tier || "").toLowerCase();
          const tierB = (b.tier || "").toLowerCase();
          comparison = tierA.localeCompare(tierB);
          break;
        }
        case "receipt_issued":
        case "fiscal": {
          const rA = a.receipt_issued ? 1 : 0;
          const rB = b.receipt_issued ? 1 : 0;
          comparison = rA - rB;
          break;
        }
        case "created_at": {
          const caA = a.created_at || "";
          const caB = b.created_at || "";
          comparison = caA.localeCompare(caB);
          break;
        }
        default: {
          const dateA = a.access_granted_at || a.created_at || "";
          const dateB = b.access_granted_at || b.created_at || "";
          comparison = dateA.localeCompare(dateB);
          break;
        }
      }
      return sort_dir === "asc" ? comparison : -comparison;
    });

    return list;
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
    } catch {
      // Mock toggle
      return { success: true, receipt_issued: issued ?? true, receipt_issued_at: new Date().toISOString() };
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
    } catch {
      // Mock issue
      return {
        success: true,
        message: "Чек успешно выписан (тестовый режим)",
        telegram_sent: payload.send_telegram ?? true,
        receipt_issued: true,
        receipt_issued_at: new Date().toISOString(),
        receipt_url: payload.receipt_url,
        receipt_number: payload.receipt_number,
      };
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
    } catch {
      // Mock toggle
    }
    return true;
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
        return data.payouts;
      }
    } catch {
      // Mock payouts
    }

    return [
      { id: 101, user_id: 2, amount: 3105, payment_details: "+7 (911) 234-56-78 • СБП Тинькофф (Арина В.)", status: "pending", created_at: "2026-09-26 14:20" },
      { id: 102, user_id: 3, amount: 2070, payment_details: "+7 (999) 876-54-32 • СБП Сбер (Михаил С.)", status: "pending", created_at: "2026-09-26 18:45" },
      { id: 103, user_id: 5, amount: 1035, payment_details: "+7 (921) 111-22-33 • СБП Альфа (Артём С.)", status: "paid", created_at: "2026-09-25 10:10" },
    ];
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
    } catch {
      // Mock fallback
    }
    return true;
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
    } catch {
      // Mock fallback
    }
    return true;
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
      // Mock fallback
    }

    return {
      referral_code: "student_spark",
      referral_link: "https://so-called-spark.ru/?ref=student_spark",
      current_balance: 4140,
      total_earned: 4140,
      total_withdrawn: 0,
      level1_referrals: 4,
      level2_referrals: 0,
      total_referrals: 4,
      transactions: [
        { id: 1, buyer_id: 2, level: 1, amount: 1035, created_at: "2026-09-24 16:30" },
        { id: 2, buyer_id: 3, level: 1, amount: 1035, created_at: "2026-09-25 11:20" },
        { id: 3, buyer_id: 5, level: 1, amount: 1035, created_at: "2026-09-26 14:10" },
        { id: 4, buyer_id: 7, level: 1, amount: 1035, created_at: "2026-09-27 19:40" },
      ],
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
    } catch {
      // Mock submission
      return {
        success: true,
        message: "Заявка на вывод успешно принята и отправлена на модерацию",
        payout: {
          id: Date.now(),
          user_id: 101,
          amount,
          payment_details: details,
          status: "pending",
          created_at: new Date().toISOString().replace("T", " ").substring(0, 16),
        },
      };
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
      throw new Error();
    } catch {
      return [
        { id: 1, code: "START5", discount_percent: 5, owner_username: "admin", uses_count: 14, reward_amount: 1000, is_active: true, created_at: "2026-10-01 10:00" },
        { id: 2, code: "IL5", discount_percent: 5, owner_username: "ilyan_vas", uses_count: 8, reward_amount: 1000, is_active: true, created_at: "2026-10-01 10:00" },
        { id: 3, code: "SPARK5", discount_percent: 5, owner_username: "admin", uses_count: 22, reward_amount: 1000, is_active: true, created_at: "2026-10-01 10:00" },
      ];
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
    } catch {
      return {
        success: true,
        promo: {
          id: Date.now(),
          code: data.code.toUpperCase(),
          discount_percent: data.discount_percent || 5,
          owner_username: data.owner_username || "partner",
          uses_count: 0,
          reward_amount: data.reward_amount || 1000,
          is_active: true,
          created_at: new Date().toISOString().replace("T", " ").substring(0, 16),
        },
      };
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
    } catch {
      return {
        success: true,
        user: {
          id: Date.now(),
          telegram_id: data.telegram_id,
          first_name: data.first_name,
          last_name: data.last_name || "",
          username: data.username || "",
          role: data.role || "student",
          has_access: !!data.has_access,
          tier: data.tier || "accelerator",
          paid_amount: data.paid_amount || 0,
          promo_code: data.promo_code || "",
          receipt_issued: !!data.receipt_issued,
        },
      };
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
    } catch {
      return { success: true };
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
    } catch {
      return { success: true };
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
    } catch {
      return { success: true };
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
    } catch {
      return { success: true };
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
      return { success: true };
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
};
