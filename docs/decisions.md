# Decisions

## 2026-11-08: accelerator price A/B experiment retired

The `pricing_price` experiment (control 10 990 ₽ vs lower 9 990 ₽) is removed from the site, the pricing API, YooKassa checkout and the Telegram bot. Everyone sees and pays the price set in the admin panel.

- Kept on purpose: the `bot_sessions.price_experiment` / `vip_price_experiment` columns and model fields (no destructive migration; old values are ignored), the `pe_lower` deep-link parsing (links already sent still open), and old `experiment_assignments` rows (the admin report still shows the historical result).
- Copy experiments `hero_cta_copy` and `pricing_cta_copy` are unchanged.
- Visitors who already saw 9 990 ₽ pay 10 990 ₽ after deploy.
