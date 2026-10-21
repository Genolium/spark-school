#!/bin/sh
# Create and verify a PostgreSQL backup before deploy/migrations.
# Aborts the deployment if the backup cannot be created or looks invalid.
set -eu

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

POSTGRES_USER="${POSTGRES_USER:-spark_user}"
POSTGRES_DB="${POSTGRES_DB:-spark_db}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
TS=$(date +%Y%m%d_%H%M%S)
TMP="${BACKUP_DIR}/predeploy_${TS}.sql"
OUT="${TMP}.gz"
COMPOSE="${DOCKER_COMPOSE:-docker compose}"

mkdir -p "$BACKUP_DIR"

if ! $COMPOSE ps -q postgres >/dev/null 2>&1 || [ -z "$($COMPOSE ps -q postgres 2>/dev/null)" ]; then
    echo "[predeploy-backup] no existing postgres container; first deploy, skipping"
    exit 0
fi

echo "[predeploy-backup] dumping ${POSTGRES_DB}..."
if ! $COMPOSE exec -T postgres pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --no-privileges > "$TMP"; then
    rm -f "$TMP"
    echo "[predeploy-backup] ERROR: pg_dump failed" >&2
    exit 1
fi

if [ ! -s "$TMP" ]; then
    rm -f "$TMP"
    echo "[predeploy-backup] ERROR: dump is empty" >&2
    exit 1
fi

gzip -c "$TMP" > "$OUT"
rm -f "$TMP"
if ! gzip -t "$OUT"; then
    rm -f "$OUT"
    echo "[predeploy-backup] ERROR: gzip integrity check failed" >&2
    exit 1
fi

if ! gzip -dc "$OUT" | grep -q 'CREATE TABLE public.users'; then
    rm -f "$OUT"
    echo "[predeploy-backup] ERROR: dump does not contain public.users" >&2
    exit 1
fi

SIZE=$(wc -c < "$OUT" | tr -d ' ')
if [ "$SIZE" -lt 1024 ]; then
    rm -f "$OUT"
    echo "[predeploy-backup] ERROR: dump is suspiciously small (${SIZE} bytes)" >&2
    exit 1
fi

sha256sum "$OUT" > "${OUT}.sha256" 2>/dev/null || shasum -a 256 "$OUT" > "${OUT}.sha256"

# Keep the last 10 pre-deploy backups.
ls -1t "${BACKUP_DIR}"/predeploy_*.sql.gz 2>/dev/null | tail -n +11 | while read -r old; do
    rm -f "$old" "$old.sha256"
done

echo "[predeploy-backup] OK: ${OUT} ($(du -h "$OUT" | cut -f1))"
echo "[predeploy-backup] checksum: $(cat "${OUT}.sha256")"
