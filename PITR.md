# Point-in-time recovery (PITR)

Daily logical backups are not PITR. For production-grade recovery:

1. Enable WAL archiving on PostgreSQL:
   ```conf
   wal_level = replica
   archive_mode = on
   archive_command = 'test ! -f /archive/%f && cp %p /archive/%f'
   ```
2. Keep at least 7 days of WAL in a separate volume or object storage.
3. Install `pgBackRest` or `wal-g` for base backups and retention.
4. Once per month run a restore drill into a separate database:
   ```bash
   ./scripts/restore-drill.sh /backups/latest.sql.gz
   ```
   For PITR, restore the base backup and replay WAL to a timestamp before the
   incident.
5. Record the drill result, restore duration and verified row counts.

The current repository ships logical backup + restore drill only; WAL
archiving must be enabled on the database host.
