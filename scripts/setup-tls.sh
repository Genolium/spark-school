#!/bin/sh
# Install an external Nginx TLS terminator in front of Caddy.
# Usage: sudo DOMAIN=so-called-spark.ru LETSENCRYPT_EMAIL=you@example.com ./scripts/setup-tls.sh
set -eu

DOMAIN="${DOMAIN:-${1:-}}"
EMAIL="${LETSENCRYPT_EMAIL:-${2:-}}"
INCLUDE_WWW="${INCLUDE_WWW:-0}"
UPSTREAM_PORT="${PORT_HTTP:-38080}"

if [ "$(id -u)" -ne 0 ]; then
    echo "run as root (sudo)" >&2
    exit 1
fi
if [ -z "$DOMAIN" ] || [ -z "$EMAIL" ]; then
    echo "DOMAIN and LETSENCRYPT_EMAIL are required" >&2
    exit 1
fi
for bin in nginx certbot; do
    if ! command -v "$bin" >/dev/null 2>&1; then
        echo "missing $bin. Install nginx and certbot first." >&2
        exit 1
    fi
done

WWW=""
CERT_ARGS="-d $DOMAIN"
if [ "$INCLUDE_WWW" = "1" ]; then
    WWW="www.$DOMAIN"
    CERT_ARGS="$CERT_ARGS -d www.$DOMAIN"
fi

echo "[tls] requesting certificate for $DOMAIN..."
# shellcheck disable=SC2086
certbot certonly --nginx $CERT_ARGS --non-interactive --agree-tos -m "$EMAIL" --keep-until-expiring

TARGET="/etc/nginx/sites-available/spark.conf"
sed -e "s|__DOMAIN__|$DOMAIN|g" -e "s|__WWW__|$WWW|g" -e "s|__UPSTREAM_PORT__|$UPSTREAM_PORT|g" ./scripts/nginx-spark.conf.template > "$TARGET"
ln -sf "$TARGET" /etc/nginx/sites-enabled/spark.conf
nginx -t
systemctl reload nginx

echo "[tls] checking https://$DOMAIN/health/live"
curl -fsS --max-time 15 "https://$DOMAIN/health/live" >/dev/null
echo "[tls] OK. Caddy remains HTTP-only on 127.0.0.1:38080."
