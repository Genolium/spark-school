# проект «так называемый SPARK» 🚀

> Полнофункциональная образовательная веб-платформа с бесшовной интеграцией в Telegram для подготовки студентов к грантовым программам академического обмена США (проект «так называемый SPARK», **INSPIRE**).

---

## 🏛 Архитектура проекта

```text
spark-school/
├── docker-compose.yml         # Оркестрация Caddy 2, Go Backend, Next.js Frontend, PostgreSQL 16
├── Caddyfile                  # HTTP Reverse Proxy (TLS — на внешнем Nginx/балансировщике)
├── .env.example               # Образец всех необходимых переменных окружения
│
├── backend/                   # Серверное приложение на Golang 1.26
│   ├── cmd/
│   │   ├── server/main.go     # Точка входа HTTP API и фонового Telegram-бота
│   │   └── bot/main.go        # Точка входа отдельного Telegram-бота
│   ├── internal/
│   │   ├── auth/              # HMAC-SHA256 валидация Telegram Login Widget, JWT-сессии
│   │   ├── bot/               # Воркер Telegram-бота (диплинки, выдача гайдов, уведомления)
│   │   ├── config/            # Конфигурация из переменных окружения
│   │   ├── database/          # Пул соединений PostgreSQL, запуск миграций (migrations.go)
│   │   │   └── migrations/    # Goose SQL-миграции схемы, применяются backend при старте
│   │   ├── diagnostics/       # Диагностический план
│   │   ├── handlers/          # HTTP эндпоинты (Auth, Affiliate, Admin, Payments, Analytics и др.)
│   │   ├── models/            # Модели сущностей БД (User, Payouts, Referrals)
│   │   ├── money/             # Денежная арифметика в копейках (int64)
│   │   ├── monitoring/        # Фоновые проверки и алерты администратору в Telegram
│   │   ├── outbox/            # Dispatcher outbox-событий (повторы, dead-letter)
│   │   ├── payments/          # Сервис платежей
│   │   ├── privacy/           # Retention аналитики и кликов (ANALYTICS_RETENTION_DAYS)
│   │   ├── readmodels/        # Отчётные выборки (например, выручка)
│   │   ├── security/          # Rate limiter (в памяти и в БД), security headers, метрики
│   │   └── services/          # Бизнес-логика (двухуровневая реферальная система 15% / 5%, YooKassa)
│   └── Dockerfile             # Multi-stage сборка Go-бинарника на Alpine
│
└── frontend/                  # Клиентское веб-приложение Next.js 16 + React 19 (App Router)
    ├── src/
    │   ├── app/
    │   │   ├── page.tsx       # Продающий лендинг (Spatial Bento-Glass, Dark Obsidian)
    │   │   ├── admin/         # Панель управления администратора (финансы, студенты, выплаты)
    │   │   ├── partner/       # Партнёрский кабинет (15%/5% реферальная сеть)
    │   │   ├── plan/[token]/  # Страница плана по токену
    │   │   ├── offer/         # Публичная оферта самозанятого (ИНН 780739313219)
    │   │   ├── privacy/       # Политика конфиденциальности по 152-ФЗ
    │   │   ├── principles/    # Страница принципов
    │   │   └── terms/         # Страница условий
    │   ├── components/
    │   │   └── Admin/         # Финансовые KPI, таблицы, выплаты, отчёты
    │   ├── context/           # Провайдер темы оформления
    │   └── lib/               # API-клиенты (api-public, api-affiliate, api-admin) и фасад api.ts
    └── Dockerfile             # Multi-stage сборка Next.js на Node 22 Alpine
```

---

## 🚀 Быстрый запуск в продакшне (1-Click Docker)

### 1. Требования к серверу (VPS)
* Ubuntu 22.04 / 24.04 LTS
* Установленные `docker` и `docker compose`
* Открытые порты `80` и `443`
* Привязанный домен `so-called-spark.ru` к IP-адресу сервера

### 2. Развёртывание
```bash
# Клонируйте репозиторий
git clone https://github.com/your-org/spark-school.git /opt/spark-school
cd /opt/spark-school

# Соберите архив деплоя: скрипт создаёт папку deploy/ и в ней deploy.sh
node scripts/package-deploy.js
cd deploy

# Создайте файл переменных окружения (deploy.sh ищет .env рядом с собой)
cp .env.example .env
nano .env

# Соберите образы и запустите контур с проверкой health
./deploy.sh
```

`deploy.sh` собирает образы **до** остановки текущей версии, ждёт healthy-статусы
backend/frontend/caddy, проверяет `/health/ready` (PostgreSQL + версия миграций)
и откатывается на предыдущие образы при ошибке.

> После `git pull` пересоздай контейнеры, которые читают конфиги через bind mount:
> `docker compose up -d --force-recreate caddy postgres_backup`.
> Иначе Docker может продолжить использовать старый inode `Caddyfile`/`backup.sh`.

> ⚠️ **TLS:** Caddy работает с `auto_https off` и слушает HTTP. Production-трафик
> обязан приходить через внешний TLS-терминатор (Nginx, cloud load balancer или
> отдельный Caddy с сертификатом). Без него сайт доступен только по HTTP.
>
> Backend в production **не запустится**, если не заданы сильные `JWT_SECRET`,
> `ADMIN_PASSWORD`, `POSTGRES_PASSWORD`, `TELEGRAM_WEBHOOK_SECRET`,
> `TELEGRAM_BOT_TOKEN` и ключ хотя бы одного платёжного провайдера.

Маршрутизация:
* `/` ➔ Next.js фронтенд (`:3000`)
* `/api/*` ➔ Go бэкенд (`:8080`)
* `/health/live`, `/health/ready` ➔ пробы backend

---

## 💻 Локальная разработка

### Фронтенд (Next.js):
```bash
cd frontend
npm ci
npm run dev # Запуск на http://localhost:3000
```
Фронтенд работает с Go API. Если API недоступен, клиент показывает пустые или ошибочные состояния, а не демо-данные.

### Бэкенд (Golang):
```bash
cd backend
go run ./cmd/server
```

---

## 🤝 Что видит студент после оплаты

Вместе со ссылкой в канал бот отправляет короткое сообщение:

1. зайти в канал по ссылке;
2. Иль напишет в личку с несколькими вопросами — ответить;
3. все задания и материалы — в канале.

Отдельной анкеты, очереди онбординга и автоматических напоминаний студенту нет:
дальнейшая коммуникация идёт с личного аккаунта.

## 📱 API для Mini App

`API.md` описывает вход через Telegram `initData` и студенческие endpoints
`/api/v1/student/me` и `/api/v1/student/orders`. Это API-first база для
будущего Mini App: отдельный клиент сможет работать с тем же JWT, платежами и
правами, без второго backend.

## 🔗 Короткие ссылки кампаний (Campaign links)

Публичные ссылки вида `https://so-called-spark.ru/r/<slug>` ведут на закрытый
канал, в Telegram-бота или на лендинг с UTM-метками. Каждый переход пишется в
`campaign_link_clicks` вместе с first-party cookie `spark_vid` и случайным
`attribution_token`.

Создание и отчёт — в админке на вкладке «Студенты и доступы»: блок
«Генератор рекламных ссылок и меток» создаёт ссылку и показывает готовый
адрес, «Отчёт по коротким ссылкам» показывает переходы, уникальных
посетителей, визиты, лиды и деньги по каждой ссылке. Деактивация сохраняет
историю переходов, удаление ссылки удаляет и её.

Как читать метрики:

* **Переходы** — каждый запрос к `/r/<slug>`, включая HEAD-запросы превью
  мессенджеров и повторные заходы одного человека, поэтому это не число людей;
* **Уникальные** — различные `visitor_id` за период; если браузер не сохраняет
  cookie, каждый заход считается новым посетителем;
* **Визиты и лиды** — считаются по каждой ссылке отдельно: посетитель,
  пришедший по двум ссылкам, учитывается в обеих, поэтому суммы строк могут
  пересекаться;
* **Деньги** — только успешные оплаты, и каждая оплата относится ровно к одной
  ссылке: к первому клику этого посетителя внутри отчётного периода, но не
  позже платежа (first-touch). Клик до начала периода оставляет оплату без
  атрибуции. Оплаты без `visitor_id` в отчёт по ссылкам не попадают.

Приватность: каждая строка клика хранит `visitor_id`, `attribution_token`, а
также `User-Agent` и `Referer`, обрезанные до 255 байт (Referer может содержать
идентифицирующий URL). Сырой IP-адрес не сохраняется никогда: колонка `ip_hash`
зарезервирована и сейчас не заполняется. Клики старше
`ANALYTICS_RETENTION_DAYS` удаляются ежедневно вместе с аналитикой, а `erase`
пользователя удаляет и его клики. После удаления по retention старые периоды
больше не могут атрибутировать деньги — такие оплаты показываются как без
атрибуции.

## 🔐 TLS и backup перед миграцией

- TLS завершается внешним Nginx: `TLS.md` и `scripts/setup-tls.sh`.
  Caddy слушает только `127.0.0.1:38080`.
- `deploy.sh` перед сборкой и миграциями запускает
  `scripts/predeploy-backup.sh`: проверяет `pg_dump`, gzip и наличие
  `public.users`. Если backup не удался — деплой останавливается, рабочая
  версия не трогается.
- Backup складывается в `./backups/predeploy_*.sql.gz` с SHA-256 и хранит
  последние 10 копий.

## 🧪 CI, ошибки и staging

- `.github/workflows/ci.yml` запускает Go race tests на PostgreSQL, frontend
  build, npm audit, e2e/contract tests, Playwright smoke и Docker build.
- `.github/workflows/uptime.yml` каждые 15 минут проверяет `/health/ready`
  снаружи и пишет в Telegram при сбое.
- `SENTRY_DSN` включает Sentry на backend; ошибки браузера отправляются на
  `/api/v1/client-errors` и попадают в Sentry без cookies и значений форм.
- Dead-letter outbox виден в админке, событие можно повторить вручную.
- `STAGING.md` описывает отдельный staging-контур.
- `PITR.md` и `scripts/restore-drill.sh` — восстановление и регулярные учения.
- `tests/load/public-api.js` — k6-нагрузочный сценарий для staging.

## 🩺 Мониторинг и приватность

Backend раз в минуту проверяет и присылает алерты администратору в Telegram:

* события outbox в статусе `failed`;
* успешные платежи без серверного `payment_confirmed` за 7 дней;
* последний успешный backup старше 24 часов;
* `/health/ready` (PostgreSQL + версия миграций);
* 5xx ответы webhook-маршрутов.

Алерты имеют cooldown 30 минут. Backup-контейнер пишет heartbeat в
`backup_heartbeat` после проверки архива.

Приватность:

* `analytics_events`, `experiment_assignments` и `campaign_link_clicks` старше
  `ANALYTICS_RETENTION_DAYS` (по умолчанию 180 дней) удаляются ежедневно;
* `POST /api/v1/admin/users/:id/erase` удаляет аналитику и клики пользователя,
  анонимизирует профиль, сохраняя финансовые операции для бухгалтерии;
* `TRUSTED_PROXIES` ограничивает, каким reverse-proxy разрешено передавать
  `X-Forwarded-For`; иначе per-IP rate limit можно обойти заголовком;
* `ADMIN_TOTP_SECRET` включает обязательный 6-значный TOTP-код при входе;
* JWT администратора живёт 12 часов и отзывается через `revoked_tokens`;
* все изменяющие действия администратора пишутся в `admin_audit_log`.

## 🔒 Безопасность и защита контента

1. **Закрытый Telegram-канал:** контент публикуется в закрытом канале с приватными пригласительными ссылками для каждого оплатившего студента.
2. **Telegram HMAC-SHA256:** подлинность данных Telegram Login Widget валидируется через секретный ключ, созданный хешированием токена бота.
3. **Защита панели администратора:** вход в `/admin` защищён паролем администратора и JWT-авторизацией.

---

## 📞 Контакты и поддержка
* **Куратор проекта:** так называемый Иль (Финалист SPARK '26, University of Wyoming)
* **Telegram-бот поддержки:** [@spark_prep_bot](https://t.me/spark_prep_bot)
* **Юридическое лицо:** Самозанятый Иль О. В. (так называемый Иль), ИНН: `780739313219`
