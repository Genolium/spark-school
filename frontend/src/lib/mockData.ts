import { User, PayoutRequest, AdminMetrics } from '@/types';

export const INITIAL_USERS: User[] = [
  {
    id: 1,
    telegram_id: 891230491,
    username: "so_called_il",
    first_name: "так называемый Иль",
    last_name: "",
    role: "admin",
    has_access: true,
    access_granted_at: "2026-09-01T00:00:00Z",
    created_at: "2026-09-01T00:00:00Z"
  },
  {
    id: 2,
    telegram_id: 481920381,
    username: "annapetrova",
    first_name: "Анна",
    last_name: "Петрова",
    role: "student",
    has_access: true,
    access_granted_at: "2026-09-15T12:00:00Z",
    created_at: "2026-09-14T08:30:00Z"
  },
  {
    id: 3,
    telegram_id: 719284910,
    username: "mikhail_dev",
    first_name: "Михаил",
    last_name: "Сидоров",
    role: "student",
    has_access: true,
    access_granted_at: "2026-09-18T15:20:00Z",
    created_at: "2026-09-18T10:00:00Z"
  },
  {
    id: 4,
    telegram_id: 619284910,
    username: "daria_k",
    first_name: "Дарья",
    last_name: "Кузнецова",
    role: "student",
    has_access: false,
    created_at: "2026-09-22T09:12:00Z"
  },
  {
    id: 5,
    telegram_id: 720194851,
    username: "timur_a",
    first_name: "Тимур",
    last_name: "Алиев",
    role: "student",
    has_access: false,
    created_at: "2026-09-25T11:05:00Z"
  }
];

export const INITIAL_PAYOUT_REQUESTS: PayoutRequest[] = [
  {
    id: 1,
    user_id: 2,
    username: "annapetrova",
    full_name: "Анна Петрова",
    amount: 3105,
    payment_details: "+7 999 123-45-67 (Т-Банк / СБП)",
    status: "pending",
    created_at: "2026-09-25T16:30:00Z"
  },
  {
    id: 2,
    user_id: 3,
    username: "mikhail_dev",
    full_name: "Михаил Сидоров",
    amount: 1035,
    payment_details: "2200 4501 8923 1144 (Сбербанк)",
    status: "pending",
    created_at: "2026-09-26T10:15:00Z"
  },
  {
    id: 3,
    user_id: 1,
    username: "alex_ambassador",
    full_name: "Алексей Смирнов",
    amount: 5175,
    payment_details: "+7 911 888-99-00 (Альфа-Банк / СБП)",
    status: "pending",
    created_at: "2026-09-26T14:45:00Z"
  },
  {
    id: 4,
    user_id: 4,
    username: "katya_spb",
    full_name: "Екатерина Соколова",
    amount: 2070,
    payment_details: "+7 905 333-22-11 (Т-Банк / СБП)",
    status: "approved",
    created_at: "2026-09-20T11:00:00Z",
    resolved_at: "2026-09-21T09:30:00Z"
  }
];

export const INITIAL_METRICS: AdminMetrics = {
  gross_volume: 289800,
  gross_volume_change: "+18% к прошлой неделе",
  active_students: 42,
  total_students: 49,
  conversion_rate: 6.4,
  pending_payouts_count: 3,
  pending_payouts_sum: 9315
};
