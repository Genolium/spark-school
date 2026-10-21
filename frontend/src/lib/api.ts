// Client API facade. Failures are surfaced as empty/error states, never fake data.
"use client";

import { publicApi } from "./api-public";
import { affiliateApi } from "./api-affiliate";
import { adminApi } from "./api-admin";

export * from "./api-types";
export { API_BASE, getAdminToken, getAuthToken } from "./api-base";

export const api = {
  ...publicApi,
  ...affiliateApi,
  ...adminApi,
};
