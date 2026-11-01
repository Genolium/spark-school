import type { CoursePricing } from "./api-types";

/** Fallback prices used only when the pricing API is unavailable. They must
 * match the shop's configured prices: basic 3 333 ₽ is a deliberate "fun"
 * price, accelerator control is 10 990 ₽ (lower arm 9 990 ₽). */
export const DEFAULT_PRICING: CoursePricing = {
  basic_price: 3333,
  basic_old_price: 5900,
  accelerator_price: 10990,
  accelerator_old_price: 15400,
  vip_price: 19990,
  vip_old_price: 28000,
  bank_name: "БСПБ (Банк «Санкт-Петербург»)",
};

export const ACCELERATOR_LIST_PRICE = 10900;
export const ACCELERATOR_LOWER_PRICE = 9990;
