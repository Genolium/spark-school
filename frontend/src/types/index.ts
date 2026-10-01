export interface User {
  id: number;
  telegram_id: number;
  username: string;
  first_name: string;
  last_name?: string;
  photo_url?: string;
  role: 'student' | 'admin';
  has_access: boolean;
  access_granted_at?: string;
  created_at: string;
}

export interface PayoutRequest {
  id: number;
  user_id: number;
  username: string;
  full_name: string;
  amount: number;
  payment_details: string;
  status: 'pending' | 'approved' | 'rejected';
  created_at: string;
  resolved_at?: string;
}

export interface AdminMetrics {
  gross_volume: number;
  gross_volume_change: string;
  active_students: number;
  total_students: number;
  conversion_rate: number;
  pending_payouts_count: number;
  pending_payouts_sum: number;
}
