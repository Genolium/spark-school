#!/bin/sh
set -e

BACKUP_DIR="/backups"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/spark_db_${TIMESTAMP}.sql.gz"

mkdir -p "${BACKUP_DIR}"

echo "[$(date -Iseconds)] Starting PostgreSQL database backup..."
TMP_FILE="${BACKUP_DIR}/spark_db_${TIMESTAMP}.sql"
PGPASSWORD="${POSTGRES_PASSWORD}" pg_dump -h "${POSTGRES_HOST:-postgres}" -U "${POSTGRES_USER:-spark_user}" -d "${POSTGRES_DB:-spark_db}" > "${TMP_FILE}"

if [ ! -s "${TMP_FILE}" ]; then
    echo "[$(date -Iseconds)] ERROR: pg_dump failed or generated an empty file!"
    rm -f "${TMP_FILE}"
    exit 1
fi

gzip -c "${TMP_FILE}" > "${BACKUP_FILE}"
rm -f "${TMP_FILE}"

if ! gzip -t "${BACKUP_FILE}"; then
    echo "[$(date -Iseconds)] ERROR: gzip archive integrity test failed!"
    rm -f "${BACKUP_FILE}"
    exit 1
fi

FILE_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
FILE_BYTES=$(wc -c < "${BACKUP_FILE}" | tr -d ' ')
echo "[$(date -Iseconds)] Backup created successfully: ${BACKUP_FILE} (${FILE_SIZE})"

# Heartbeat for the backend monitoring worker. Keep financial/backup data even
# if this write fails, but the stale heartbeat will raise an alert.
PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST:-postgres}" -U "${POSTGRES_USER:-spark_user}" -d "${POSTGRES_DB:-spark_db}" -v ON_ERROR_STOP=1 <<SQL || echo "[WARN] Failed to write backup heartbeat"
CREATE TABLE IF NOT EXISTS backup_heartbeat (
    id INT PRIMARY KEY,
    last_success_at TIMESTAMP WITH TIME ZONE NOT NULL,
    file_name VARCHAR(255),
    size_bytes BIGINT
);
INSERT INTO backup_heartbeat (id, last_success_at, file_name, size_bytes)
VALUES (1, NOW(), '${BACKUP_FILE}', ${FILE_BYTES})
ON CONFLICT (id) DO UPDATE SET last_success_at = NOW(), file_name = EXCLUDED.file_name, size_bytes = EXCLUDED.size_bytes;
SQL

# Retain backups for 14 days, remove older dumps
find "${BACKUP_DIR}" -name "spark_db_*.sql.gz" -type f -mtime +14 -delete
echo "[$(date -Iseconds)] Old backups cleaned up (retention: 14 days)."

# Optional: Notification / Upload to S3 or Telegram Archive Channel if credentials configured
if [ -n "${BACKUP_TELEGRAM_BOT_TOKEN}" ] && [ -n "${BACKUP_TELEGRAM_CHAT_ID}" ]; then
    echo "[$(date -Iseconds)] Sending backup copy to Telegram Archive..."
    TG_API_BASE="${TELEGRAM_API_SERVER:-https://api.telegram.org}"
    TG_API_BASE="${TG_API_BASE%/}"
    HTTP_CODE=$(curl -s -o /tmp/backup-telegram-response.json -w "%{http_code}" \
            -F chat_id="${BACKUP_TELEGRAM_CHAT_ID}" \
            -F document=@"${BACKUP_FILE}" \
            -F caption="📦 Резервная копия БД: spark_db_${TIMESTAMP} (${FILE_SIZE})" \
            "${TG_API_BASE}/bot${BACKUP_TELEGRAM_BOT_TOKEN}/sendDocument" || echo "000")
    if [ "$HTTP_CODE" != "200" ]; then
        echo "[WARN] Telegram backup upload failed with HTTP ${HTTP_CODE}."
    fi
fi

echo "[$(date -Iseconds)] Backup job completed successfully."
