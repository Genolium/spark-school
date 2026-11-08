#!/usr/bin/env python3
"""Проверяет тексты агентов на AI-штампы, запрещённые обещания и старые цены.

Запуск:
    python3 scripts/validate_output.py --changed
    python3 scripts/validate_output.py drafts/post-2026-11-08.md
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

AI_STAMPS = [
    "в современном мире",
    "в эпоху цифровизации",
    "давайте разберёмся",
    "давайте разберемся",
    "не секрет, что",
    "погрузимся в",
    "важно отметить",
    "стоит подчеркнуть",
    "подводя итог",
    "ключевой аспект",
    "инновационный подход",
]

# These need word boundaries: plain substring "не просто" matches "мне просто".
AI_STAMP_PATTERNS = [
    r"\bне просто\b",
    r"\bэто не просто\b",
]

FORBIDDEN_CLAIMS = [
    "гарантируем грант",
    "гарантированный грант",
    "100% поступление",
    "100% грант",
    "точно выиграешь",
    "точно поступишь",
    "комиссия обязана",
    "напишем эссе за вас",
    "сделаем заявку за вас",
    "осталось 2 места",
    "осталось два места",
    "скидка сгорит через час",
]

STALE_PRICES = [
    "6 900",
    "6900",
    "14 900",
    "14900",
    "3 990",
    "3990",
]

SECRET_PATTERNS = [
    r"TELEGRAM_BOT_TOKEN\s*=\s*\S+",
    r"JWT_SECRET\s*=\s*\S+",
    r"POSTGRES_PASSWORD\s*=\s*\S+",
    r"YOOKASSA_SECRET_KEY\s*=\s*\S+",
    r"\b\d{8,10}:[A-Za-z0-9_-]{30,}\b",  # Telegram bot token shape
]

TEXT_SUFFIXES = {".md", ".txt", ".html", ".json", ".yaml", ".yml"}
TARGET_DIRS = ("drafts", "research", "output")


def changed_files() -> list[Path]:
    try:
        root = Path(
            subprocess.check_output(
                ["git", "rev-parse", "--show-toplevel"],
                text=True,
                stderr=subprocess.DEVNULL,
            ).strip()
        )
    except Exception:
        return []

    paths: list[str] = []
    try:
        paths += subprocess.check_output(
            ["git", "-C", str(root), "diff", "--name-only", "--diff-filter=ACMR"],
            text=True,
            stderr=subprocess.DEVNULL,
        ).splitlines()
        paths += subprocess.check_output(
            ["git", "-C", str(root), "ls-files", "--others", "--exclude-standard"],
            text=True,
            stderr=subprocess.DEVNULL,
        ).splitlines()
    except Exception:
        return []

    files: list[Path] = []
    for line in paths:
        path = (root / line.strip()).resolve()
        if path.suffix.lower() not in TEXT_SUFFIXES:
            continue
        if not any(part in TARGET_DIRS for part in path.parts):
            continue
        files.append(path)
    return files


def scan(path: Path) -> list[str]:
    if not path.exists() or path.suffix.lower() not in TEXT_SUFFIXES:
        return []
    text = path.read_text(encoding="utf-8", errors="replace")
    lower = text.lower()
    problems: list[str] = []

    for phrase in AI_STAMPS:
        if phrase in lower:
            problems.append(f"AI-штамп: «{phrase}»")

    for pattern in AI_STAMP_PATTERNS:
        if re.search(pattern, lower):
            problems.append(f"AI-штамп: «{pattern}»")

    for phrase in FORBIDDEN_CLAIMS:
        if phrase in lower:
            problems.append(f"Запрещённое обещание: «{phrase}»")

    for price in STALE_PRICES:
        if price in text:
            problems.append(
                f"Похоже на старую цену «{price}». Проверьте /api/v1/pricing и админку."
            )

    for pattern in SECRET_PATTERNS:
        if re.search(pattern, text):
            problems.append("Похоже на секрет или токен в тексте. Уберите его.")

    if "http" not in lower and path.suffix == ".md" and "drafts" in path.parts:
        problems.append("В черновике нет ссылки. Добавьте один целевой CTA.")

    return problems


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("files", nargs="*")
    parser.add_argument("--changed", action="store_true")
    args = parser.parse_args()

    files = [Path(p) for p in args.files]
    if args.changed:
        files = changed_files()

    failed = False
    for path in files:
        problems = scan(path)
        if problems:
            failed = True
            print(f"\n{path}:")
            for problem in problems:
                print(f"  ✗ {problem}")

    if failed:
        print("\nПроверка не пройдена. Исправьте текст и повторите.")
        return 1

    print("Проверка пройдена.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
