import type { CoursePricing } from "./api-types";

/** Fallback prices used only when the pricing API is unavailable. They must
 * match the shop's configured prices: basic 3 333 ₽ is a deliberate "fun"
 * price, accelerator is 9 990 ₽. */
export const DEFAULT_PRICING: CoursePricing = {
  basic_price: 3333,
  basic_old_price: 5900,
  accelerator_price: 9990,
  accelerator_old_price: 15400,
  vip_price: 15990,
  vip_old_price: 20000,
  basic_enabled: true,
  accelerator_enabled: true,
  vip_enabled: true,
  bank_name: "БСПБ (Банк «Санкт-Петербург»)",
};
