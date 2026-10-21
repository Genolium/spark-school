# External TLS

The compose stack intentionally does **not** terminate TLS. Caddy listens on
`127.0.0.1:${PORT_HTTP:-38080}` and an external Nginx/load balancer terminates
HTTPS.

## One-time setup on a VPS

1. Point DNS `A/AAAA` records for the domain to the server.
2. Install Nginx and Certbot:
   ```bash
   apt update && apt install -y nginx certbot python3-certbot-nginx
   ```
3. From the unpacked deploy directory:
   ```bash
   sudo DOMAIN=so-called-spark.ru \
        LETSENCRYPT_EMAIL=vas.ilyan@icloud.com \
        INCLUDE_WWW=0 \
        sh ./scripts/setup-tls.sh
   ```
4. Verify:
   ```bash
   curl -fsS https://so-called-spark.ru/health/ready
   curl -I https://so-called-spark.ru/
   ```
5. Renewal is handled by the system Certbot timer. Check:
   ```bash
   systemctl list-timers | grep certbot
   ```

## If the load balancer is remote

Binding Caddy to `127.0.0.1` only works when Nginx is on the same host. For a
remote cloud load balancer, change the compose port to a private interface and
firewall it to the balancer IPs.

## Trusted proxies

`TRUSTED_PROXIES` must include the address Caddy sees for Nginx. In the default
Docker setup this is the bridge network (`172.16.0.0/12`). Never add public
client ranges.

## Rollback

Nginx config: `/etc/nginx/sites-available/spark.conf`. Keep a copy before
changes. Certificates are under `/etc/letsencrypt/live/<domain>/`.
