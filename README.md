# проект «так называемый SPARK» 🚀

> Полнофункциональная образовательная веб-платформа с бесшовной интеграцией в Telegram для подготовки студентов к грантовым программам академического обмена США (проект «так называемый SPARK», **INSPIRE**).

---

## 🏛 Архитектура проекта

```text
spark-school/
├── docker-compose.yml         # Оркестрация Caddy 2, Go Backend, Next.js Frontend, PostgreSQL 16
├── Caddyfile                  # Авто-SSL, Reverse Proxy, сжатие gzip/zstd, заголовки безопасности
├── .env.example               # Образец всех необходимых переменных окружения
├── seed.sql                   # SQL-дамп со схемой базы данных и индексами
│
├── backend/                   # Серверное приложение на Golang 1.22+
│   ├── cmd/server/main.go     # Точка входа HTTP API и фонового Telegram-бота
│   ├── internal/
│   │   ├── auth/telegram.go   # HMAC-SHA256 валидация Telegram Login Widget, JWT-сессии
│   │   ├── bot/bot.go         # Воркер Telegram-бота (диплинки, выдача гайдов, уведомления)
│   │   ├── database/db.go     # Пул соединений PostgreSQL и GORM миграции
│   │   ├── handlers/          # HTTP эндпоинты (Auth, Affiliate, Admin)
│   │   ├── models/models.go   # Модели сущностей БД (User, Payouts, Referrals)
│   │   └── services/          # Бизнес-логика (двухуровневая реферальная система 15% / 5%)
│   └── Dockerfile             # Multi-stage сборка Go-бинарника на Alpine
│
└── frontend/                  # Клиентское веб-приложение Next.js 14+ (App Router)
    ├── src/
    │   ├── app/
    │   │   ├── page.tsx       # Продающий лендинг (Spatial Bento-Glass, Dark Obsidian)
    │   │   ├── admin/         # Панель управления администратора (финансы, студенты, выплаты)
    │   │   ├── partner/       # Партнёрский кабинет (15%/5% реферальная сеть)
    │   │   ├── offer/         # Публичная оферта самозанятого (ИНН 780739313219)
    │   │   └── privacy/       # Политика конфиденциальности по 152-ФЗ
    │   ├── components/
    │   │   └── Admin/         # Финансовые KPI, таблица студентов, выплаты
    │   ├── context/           # Провайдер сессии пользователя и демо-свитчер
    │   └── lib/api.ts         # Клиент к Go API с автономным mock-фоллбэком
    └── Dockerfile             # Multi-stage сборка Next.js на Node 20 Alpine
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

# Создайте файл переменных окружения
cp .env.example .env
nano .env

# Запустите весь контур одной командой
docker compose up -d --build
```

Caddy автоматически выпустит бесплатный SSL-сертификат Let's Encrypt и распределит трафик:
* `https://so-called-spark.ru` ➔ Next.js фронтенд (`:3000`)
* `https://so-called-spark.ru/api/*` ➔ Go бэкенд (`:8080`)

---

## 💻 Локальная разработка

### Фронтенд (Next.js):
```bash
cd frontend
pnpm install
pnpm dev # Запуск на http://localhost:3000
```
Фронтенд автоматически поддерживает автономный режим работы с демо-профилями (`Студент с доступом`, `Гость`, `Администратор`).

### Бэкенд (Golang):
```bash
cd backend
go run ./cmd/server
```

---

## 🔒 Безопасность и защита контента

1. **Закрытый Telegram-канал:** контент публикуется в закрытом канале с приватными пригласительными ссылками для каждого оплатившего студента.
2. **Telegram HMAC-SHA256:** подлинность данных Telegram Login Widget валидируется через секретный ключ, созданный хешированием токена бота.
3. **Защита панели администратора:** вход в `/admin` защищён паролем администратора и JWT-авторизацией.

---

## 📞 Контакты и поддержка
* **Куратор проекта:** так называемый Иль (Финалист SPARK '26, University of Wyoming)
* **Telegram-бот поддержки:** [@spark_prep_bot](https://t.me/spark_prep_bot)
* **Юридическое лицо:** Самозанятый Иль О. В. (так называемый Иль), ИНН: `780739313219`
