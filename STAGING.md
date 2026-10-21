# Staging contour

Staging must use a separate database and must never send real payment webhooks
to production.

1. Copy `.env.example` to `.env.staging` and fill staging-only secrets.
2. Use a separate Compose project and volumes:
   ```bash
   COMPOSE_PROJECT_NAME=spark-staging \
   docker compose --env-file .env.staging -p spark-staging up -d --build
   ```
3. Point `DOMAIN`, `FRONTEND_URL`, `TELEGRAM_WEBHOOK_SECRET` and payment
   provider callbacks to staging.
4. Run migrations and smoke tests:
   ```bash
   curl -fsS http://127.0.0.1:38088/health/ready
   npm run build
   node --test tests/e2e_verification.test.mjs tests/api_contract.test.mjs
   ```
5. Use a separate Telegram bot token and a separate YooKassa/Tribute test
   account. Never reuse production secrets.

Known limitation: the base compose file publishes the same ports. Override
`PORT_HTTP`, `PORT_BACKEND`, `PORT_FRONTEND`, `PORT_POSTGRES` in `.env.staging`.
