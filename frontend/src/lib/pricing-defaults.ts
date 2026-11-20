import type { CoursePricing } from "./api-types";

/** Default close time for Stage 1 sales: 21 November 2026 00:00:00 MSK (UTC+3). */
export const DEFAULT_SALES_CLOSE_AT = "2026-11-21T00:00:00+03:00";

/** Fallback prices used only when the pricing API is unavailable. They must
 * match the shop's configured prices: basic 3 333 ₽ is a deliberate "fun"
 * price, accelerator is 9 990 ₽. */
export const DEFAULT_PRICING: CoursePricing = {
  basic_price: 3333,
  basic_old_price: 5900,
  basic_feedback_price: 5000,
  basic_feedback_old_price: 6900,
  accelerator_price: 9990,
  accelerator_old_price: 15400,
  vip_price: 15990,
  vip_old_price: 20000,
  basic_enabled: true,
  basic_feedback_enabled: true,
  accelerator_enabled: true,
  vip_enabled: true,
  sales_close_at: DEFAULT_SALES_CLOSE_AT,
  sales_closed: false,
  bank_name: "БСПБ (Банк «Санкт-Петербург»)",
};

/** Checks whether all new sales are closed (by server flag, timer, or all toggles off). */
export function isPricingSalesClosed(pricing: CoursePricing, nowMs: number = Date.now()): boolean {
  if (pricing.sales_closed) return true;
  if (pricing.sales_close_at) {
    const closeMs = new Date(pricing.sales_close_at).getTime();
    if (!Number.isNaN(closeMs) && nowMs >= closeMs) {
      return true;
    }
  }
  const anyEnabled =
    pricing.basic_enabled !== false ||
    pricing.basic_feedback_enabled !== false ||
    pricing.accelerator_enabled !== false ||
    pricing.vip_enabled !== false;
  return !anyEnabled;
}
