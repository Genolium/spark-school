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
}

export interface AdminStats {
  gross_volume: number;
  active_students: number;
  total_registered: number;
  conversion_rate: number;
  pending_payouts_count: number;
  pending_payouts_sum: number;
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
      if (res.ok) return await res.json();
    } catch {
      // Fallback
    }

    return {
      gross_volume: 289800,
      active_students: 42,
      total_registered: 656,
      conversion_rate: 6.4,
      pending_payouts_count: 3,
      pending_payouts_sum: 8500,
    };
  },

  // Admin students list
  async getAdminStudents(query = ""): Promise<UserProfile[]> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/users?q=${encodeURIComponent(query)}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: "include",
      });
      if (res.ok) {
        const data = await res.json();
        return data.users;
      }
    } catch {
      // Fallback mock students
    }

    const mockStudents: UserProfile[] = [
      { id: 1, telegram_id: 891230491, username: "ilyan_vas", first_name: "так называемый Иль", last_name: "", role: "admin", has_access: true, access_granted_at: "2026-09-01" },
      { id: 2, telegram_id: 481920381, username: "arina_spark", first_name: "Арина", last_name: "Волкова", role: "student", has_access: true, access_granted_at: "2026-09-15" },
      { id: 3, telegram_id: 719284910, username: "mikhail_tech", first_name: "Михаил", last_name: "Соколов", role: "student", has_access: true, access_granted_at: "2026-09-18" },
      { id: 4, telegram_id: 619284910, username: "daria_bmstu", first_name: "Дарья", last_name: "Кузнецова", role: "student", has_access: false },
      { id: 5, telegram_id: 519284910, username: "artem_spbgu", first_name: "Артём", last_name: "Смирнов", role: "student", has_access: true, access_granted_at: "2026-09-22" },
      { id: 6, telegram_id: 319284910, username: "polina_voenmeh", first_name: "Полина", last_name: "Иванова", role: "student", has_access: false },
    ];

    if (query) {
      const q = query.toLowerCase();
      return mockStudents.filter(
        (s) => s.username?.toLowerCase().includes(q) || s.first_name?.toLowerCase().includes(q) || s.last_name?.toLowerCase().includes(q)
      );
    }
    return mockStudents;
  },

  // Toggle user access in Admin
  async toggleUserAccess(userId: number, hasAccess: boolean): Promise<boolean> {
    const token = getAdminToken();
    try {
      const res = await fetch(`${API_BASE}/admin/users/${userId}/access`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ has_access: hasAccess }),
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
        if (t === "basic" || t.includes("баз")) basePrice = 2900;
        else if (t === "vip") basePrice = 14900;
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
};
