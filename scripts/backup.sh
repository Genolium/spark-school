#!/bin/sh
set -e

BACKUP_DIR="/backups"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/spark_db_${TIMESTAMP}.sql.gz"

mkdir -p "${BACKUP_DIR}"

echo "[$(date -Iseconds)] Starting PostgreSQL database backup..."
PGPASSWORD="${POSTGRES_PASSWORD}" pg_dump -h "${POSTGRES_HOST:-postgres}" -U "${POSTGRES_USER:-spark_user}" -d "${POSTGRES_DB:-spark_db}" | gzip > "${BACKUP_FILE}"

FILE_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
echo "[$(date -Iseconds)] Backup created successfully: ${BACKUP_FILE} (${FILE_SIZE})"

# Retain backups for 14 days, remove older dumps
find "${BACKUP_DIR}" -name "spark_db_*.sql.gz" -type f -mtime +14 -delete
echo "[$(date -Iseconds)] Old backups cleaned up (retention: 14 days)."

# Optional: Notification / Upload to S3 or Telegram Archive Channel if credentials configured
if [ -n "${BACKUP_TELEGRAM_BOT_TOKEN}" ] && [ -n "${BACKUP_TELEGRAM_CHAT_ID}" ]; then
    echo "[$(date -Iseconds)] Sending backup copy to Telegram Archive..."
    curl -s -F chat_id="${BACKUP_TELEGRAM_CHAT_ID}" \
            -F document=@"${BACKUP_FILE}" \
            -F caption="📦 Резервная копия БД: spark_db_${TIMESTAMP} (${FILE_SIZE})" \
            "https://api.telegram.org/bot${BACKUP_TELEGRAM_BOT_TOKEN}/sendDocument" > /dev/null || echo "[WARN] Failed sending backup to Telegram."
fi

echo "[$(date -Iseconds)] Backup job completed successfully."
