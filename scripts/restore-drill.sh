#!/bin/sh
# Restore the latest (or given) backup into a temporary database and verify it.
set -eu

BACKUP_FILE="${1:-$(ls -1t /backups/spark_db_*.sql.gz 2>/dev/null | head -n1 || true)}"
if [ -z "${BACKUP_FILE}" ] || [ ! -f "${BACKUP_FILE}" ]; then
    echo "ERROR: backup file not found"
    exit 1
fi

PGHOST="${POSTGRES_HOST:-postgres}"
PGPORT="${POSTGRES_PORT:-5432}"
PGUSER="${POSTGRES_USER:-spark_user}"
export PGPASSWORD="${POSTGRES_PASSWORD:-}"
DRILL_DB="spark_restore_drill_$(date +%Y%m%d_%H%M%S)_$$"

cleanup() {
    dropdb -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" --if-exists "$DRILL_DB" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

psql_drill() {
    psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$DRILL_DB" -v ON_ERROR_STOP=1 -tAc "$1"
}

echo "[$(date -Iseconds)] Restoring $BACKUP_FILE into $DRILL_DB..."
createdb -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" "$DRILL_DB"
gzip -dc "$BACKUP_FILE" | psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$DRILL_DB" -v ON_ERROR_STOP=1 >/dev/null

USERS=$(psql_drill "SELECT COUNT(*) FROM users")
PAYMENTS=$(psql_drill "SELECT COUNT(*) FROM payment_transactions")
ACTIVE_USERS=$(psql_drill "SELECT COUNT(*) FROM users WHERE has_access IS TRUE")

if [ "${USERS:-0}" -le 0 ]; then
    echo "ERROR: restored database has no users"
    exit 1
fi

echo "[$(date -Iseconds)] [restore-drill] OK: users=$USERS payments=$PAYMENTS active_users=$ACTIVE_USERS"
