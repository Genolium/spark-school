import { NextRequest, NextResponse } from "next/server";

// Per-request nonce CSP. Next.js applies the nonce to its own scripts when
// the CSP header is present on the request.
export function middleware(request: NextRequest) {
  const nonce = Buffer.from(crypto.randomUUID()).toString("base64");
  const isDev = process.env.NODE_ENV !== "production";
  const connect = [
    "'self'",
    "https://so-called-spark.ru",
    "https://*.so-called-spark.ru",
    "https://api.yookassa.ru",
    "https://yookassa.ru",
    "https://api.telegram.org",
    "https://mc.yandex.ru",
    // Local development and local production smoke tests only.
    "http://localhost:*",
    "http://127.0.0.1:*",
  ];

  const csp = [
    "default-src 'self'",
    "base-uri 'self'",
    "object-src 'none'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${isDev ? " 'unsafe-eval'" : ""} https://telegram.org https://yookassa.ru https://*.yookassa.ru https://mc.yandex.ru`,
    "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
    "font-src 'self' data: https://fonts.gstatic.com",
    "img-src 'self' data: https:",
    "frame-src 'self' https://www.youtube.com https://www.youtube-nocookie.com https://telegram.org https://oauth.telegram.org https://yookassa.ru https://*.yookassa.ru",
    "frame-ancestors 'self'",
    "form-action 'self' https://yookassa.ru https://*.yookassa.ru",
    `connect-src ${connect.join(" ")}`,
  ];
  if (!isDev) csp.push("upgrade-insecure-requests");

  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-nonce", nonce);
  requestHeaders.set("content-security-policy", csp.join("; "));
  const response = NextResponse.next({ request: { headers: requestHeaders } });
  response.headers.set("Content-Security-Policy", csp.join("; "));
  return response;
}

export const config = {
  matcher: [
    "/((?!api|_next/static|_next/image|favicon.ico|icon.svg|robots.txt|sitemap.xml|opengraph-image).*)",
  ],
};
