import type { CoursePricing } from "./api-types";

/** Canonical list prices. Accelerator control is 10 900 ₽; the "lower" experiment is 9 990 ₽. */
export const DEFAULT_PRICING: CoursePricing = {
  basic_price: 3990,
  basic_old_price: 5900,
  accelerator_price: 10900,
  accelerator_old_price: 10900,
  vip_price: 14900,
  vip_old_price: 19900,
  bank_name: "БСПБ (Банк «Санкт-Петербург»)",
};

export const ACCELERATOR_LIST_PRICE = 10900;
export const ACCELERATOR_LOWER_PRICE = 9990;
