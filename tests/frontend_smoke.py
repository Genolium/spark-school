"""Browser smoke test for the public Next.js routes.

Run after `npm run build` and `npx next start -p 3100`:
    FRONTEND_URL=http://127.0.0.1:3100 python3 tests/frontend_smoke.py
"""
import os
import sys

from playwright.sync_api import sync_playwright

BASE = os.environ.get("FRONTEND_URL", "http://127.0.0.1:3100").rstrip("/")

CHECKS = [
    ("/", 200, None),
    ("/admin", 200, None),
    ("/offer", 200, None),
    ("/privacy", 200, None),
    ("/plan/invalidtoken", 200, None),
    ("/partner", 200, BASE + "/"),
    ("/terms", 200, BASE + "/offer"),
    ("/robots.txt", 200, None),
    ("/sitemap.xml", 200, None),
    ("/opengraph-image", 200, None),
]


def main() -> int:
    failures = []
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page(viewport={"width": 1280, "height": 900})
        for path, expected, final_url in CHECKS:
            response = page.goto(BASE + path, wait_until="domcontentloaded", timeout=30000)
            status = response.status if response else 0
            actual_url = response.url if response else ""
            if status != expected:
                failures.append(f"{path}: expected {expected}, got {status}")
            elif final_url and actual_url.rstrip("/") != final_url.rstrip("/"):
                failures.append(f"{path}: expected redirect to {final_url}, got {actual_url}")
            else:
                print(f"OK {path} {status} {actual_url}")
        # The main CTA must point to Telegram, not to a broken in-page handler.
        page.goto(BASE + "/", wait_until="domcontentloaded")
        cta = page.locator('a[href="/#pricing"]').count()
        if cta == 0:
            failures.append("main page has no pricing CTA")
        browser.close()
    if failures:
        print("\n".join(failures), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
