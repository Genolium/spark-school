import type { CoursePricing } from "./api-types";

/** Fallback prices used only when the pricing API is unavailable. They must
 * match the shop's configured prices: basic 3 333 ₽ is a deliberate "fun"
 * price, accelerator is 10 990 ₽. Keep in sync with the backend defaults in
 * backend/internal/models/pricing.go (checked by a Go test).
 * `satisfies` keeps every field a concrete number for callers. */
export const DEFAULT_PRICING = {
  basic_price: 3333,
  basic_old_price: 5900,
  accelerator_price: 10990,
  accelerator_old_price: 15400,
  vip_price: 19990,
  vip_old_price: 28000,
  bank_name: "БСПБ (Банк «Санкт-Петербург»)",
} satisfies CoursePricing;
