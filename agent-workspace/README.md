# agent-workspace

Рабочая папка агентов проекта «так называемый SPARK». Не входит в production-сборку.

## Быстрый старт

```bash
cd agent-workspace
claude   # или codex / opencode
```

В начале сессии:

```text
Прочитай AGENTS.md и memory/. Дальше работай по нужному скиллу.
```

## Что заполнить в первую очередь

1. `memory/style_rules.md` — 3–5 реальных постов Ильи.
2. `memory/unpacking_results.md` — интервью с Агентом 0.
3. `research/content_plan.md` — темы на месяц.
4. `memory/offers.md` — сверить цены с админкой (справочно, истина там; последняя сверка 8 ноября 2026).

## Файлы инструкций

`AGENTS.md` — единственный источник правил. `CLAUDE.md` только подключает его строкой `@AGENTS.md`, поэтому править нужно `AGENTS.md`.

## Что в git

В репозитории: `memory/`, `research/`, `drafts/`, `scripts/`, `.claude/`. Не попадает: `output/`, `media_in/`, `media_out/`, `.env`.

## Проверка текста

```bash
python3 scripts/validate_output.py drafts/post-2026-11-08.md
```

## Важно

- Не запускать второго Telegram-бота с production-токеном: он перехватит апдейты у основного.
- Не публиковать без человека.
- Не обещать грант.
- Не вставлять в промпты `.env`, чеки и персональные данные.
