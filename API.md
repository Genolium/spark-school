# API для студенческого Mini App

Все ответы — JSON. Авторизация — Bearer JWT, тот же токен, что у веб-API.

## 1. Вход через Telegram Mini App

```http
POST /api/v1/student/auth
Content-Type: application/json

{"init_data": "<window.Telegram.WebApp.initData>"}
```

Сервер проверяет подпись `initData` через `HMAC-SHA256(bot_token, "WebAppData")`,
проверяет `auth_date` (не старше 24 часов), создаёт/находит пользователя и
возвращает:

```json
{
  "token": "jwt",
  "user": {
    "telegram_id": 123,
    "username": "student",
    "first_name": "Имя",
    "has_access": true,
    "tier": "accelerator",
    "paid_amount": 6900,
    "audit_credits": 0
  }
}
```

## 2. Профиль

```http
GET /api/v1/student/me
Authorization: Bearer <jwt>
```

## 3. История платежей

```http
GET /api/v1/student/orders
Authorization: Bearer <jwt>
```

Возвращает только безопасные поля: `payment_id`, `tier`, `amount`, `status`,
`created_at`. Внутренние gateway-данные и PII не отдаются.

## Границы

- API не дублирует учебный контур: он выключен продуктово.
- Платежи, чеки и выдача доступа остаются в существующих сервисах.
- Mini App должен быть отдельным клиентом этого API, а не новым backend.
