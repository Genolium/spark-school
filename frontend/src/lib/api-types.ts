// Shared API types.

// Client API service. Failures are surfaced as empty/error states, never fake data.

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


export interface FunnelStepMetric {
  step: string;
  count: number;
  conversion_from_previous: number;
}

export interface ExperimentMetric {
  experiment: string;
  variant: string;
  visitors: number;
  bot_leads: number;
  orders: number;
  payers: number;
  revenue: number;
  conversion: number;
  uplift_percent: number;
  p_value: number;
  significant: boolean;
  min_sample_reached: boolean;
  required_sample_per_variant: number;
}

export interface DeadOutboxEvent {
  id: number;
  event_type: string;
  status: string;
  retry_count: number;
  max_retries: number;
  last_error?: string;
  created_at: string;
}

export interface FunnelData {
  filters: { from: string; to: string; source: string; campaign: string; tier: string; promo: string };
  steps: FunnelStepMetric[];
  payments: { count: number; revenue: number; average: number; source: string };
}

