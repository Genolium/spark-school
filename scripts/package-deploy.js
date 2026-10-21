/**
 * проект «так называемый SPARK» — Production Deployment Packager
 * 
 * Creates a clean, minimal, production-ready deploy folder and zip archive
 * excluding node_modules, build caches, test files, and unused temporary assets.
 */

const fs = require('fs');
const path = require('path');
const cp = require('child_process');

const ROOT_DIR = path.resolve(__dirname, '..');
const OUT_DIR = path.join(ROOT_DIR, 'deploy');
const ZIP_PATH = path.join(ROOT_DIR, 'spark-deploy.zip');

console.log('====================================================');
console.log('📦  проект «так называемый SPARK» — Packaging for Deploy');
console.log('====================================================\n');

// 1. Clean previous deploy folder and zip
if (fs.existsSync(OUT_DIR)) {
  console.log('🧹 Cleaning previous deploy/ directory...');
  try {
    fs.rmSync(OUT_DIR, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
  } catch (err) {
    if (process.platform === 'win32') {
      try {
        cp.execSync(`powershell -NoProfile -Command "Remove-Item -LiteralPath '${OUT_DIR}' -Recurse -Force"`, { stdio: 'ignore' });
      } catch (e) {
        // ignore and proceed
      }
    }
  }
}
if (fs.existsSync(ZIP_PATH)) {
  try {
    fs.rmSync(ZIP_PATH, { force: true, maxRetries: 3, retryDelay: 100 });
  } catch (err) {
    // ignore
  }
}

fs.mkdirSync(OUT_DIR, { recursive: true });

function copyFileSafe(srcRel, destRel) {
  const src = path.join(ROOT_DIR, srcRel);
  const dest = path.join(OUT_DIR, destRel || srcRel);
  if (fs.existsSync(src)) {
    fs.mkdirSync(path.dirname(dest), { recursive: true });
    fs.copyFileSync(src, dest);
  } else {
    console.warn(`⚠️ Warning: ${srcRel} does not exist`);
  }
}

function copyDir(srcAbsolute, destAbsolute, filterFn) {
  if (!fs.existsSync(srcAbsolute)) return;

  fs.mkdirSync(destAbsolute, { recursive: true });
  const entries = fs.readdirSync(srcAbsolute, { withFileTypes: true });

  for (const entry of entries) {
    const srcPath = path.join(srcAbsolute, entry.name);
    const destPath = path.join(destAbsolute, entry.name);
    const relPath = path.relative(ROOT_DIR, srcPath).replace(/\\/g, '/');

    if (filterFn && !filterFn(srcPath, relPath, entry)) {
      continue;
    }

    if (entry.isDirectory()) {
      copyDir(srcPath, destPath, filterFn);
    } else if (entry.isFile()) {
      fs.copyFileSync(srcPath, destPath);
    }
  }
}

// 2. Copy Root Configuration Files
console.log('📋 Copying root orchestration files...');
const rootFiles = [
  'docker-compose.yml',
  'Caddyfile',
  'seed.sql',
  '.env.example',
  'README.md',
  'MAIN.MD',
  'STAGING.md',
  'PITR.md',
  'API.md'
];
rootFiles.forEach(f => copyFileSafe(f));

// 2.1 Copy Deployment Scripts
console.log('📜 Copying deployment scripts...');
copyFileSafe('scripts/backup.sh');
copyFileSafe('scripts/restore-drill.sh');

// (Do not copy .env to deploy folder - server keeps its own .env or uses .env.example)

// 3. Copy Backend
console.log('⚙️ Copying Go backend files (excluding tests & local binaries)...');
copyDir(path.join(ROOT_DIR, 'backend'), path.join(OUT_DIR, 'backend'), (fullPath, relPath, entry) => {
  if (entry.name === 'node_modules' || entry.name === '.git' || entry.name === 'tmp' || entry.name === 'bin') {
    return false;
  }
  if (entry.isFile() && (entry.name.endsWith('_test.go') || entry.name.endsWith('.exe'))) {
    return false;
  }
  return true;
});

// 4. Copy Frontend
console.log('🎨 Copying Next.js frontend files (excluding node_modules & caches)...');
// Configs & manifests
const frontendConfigs = [
  'frontend/Dockerfile',
  'frontend/package.json',
  'frontend/package-lock.json',
  'frontend/next.config.mjs',
  'frontend/tailwind.config.ts',
  'frontend/postcss.config.js',
  'frontend/tsconfig.json',
  'frontend/next-env.d.ts',
  'frontend/.gitignore'
];
frontendConfigs.forEach(f => copyFileSafe(f));

// Source files
copyDir(path.join(ROOT_DIR, 'frontend/src'), path.join(OUT_DIR, 'frontend/src'), (fullPath, relPath, entry) => {
  if (entry.isFile() && (entry.name.endsWith('.bak') || entry.name.endsWith('.tmp') || entry.name.endsWith('~'))) {
    return false;
  }
  return true;
});

// Public assets — include required web assets, fonts, logos; exclude massive unused files
console.log('🖼️ Copying optimized public assets...');

// Files to explicitly exclude from public (massive unused generation artifacts >15MB)
const excludedPublicPatterns = [
  /_upscaled_4x/i,
  /^thumb_/i,
  /spatial-campus-hero1/i,
  /spatial-campus-hero3/i,
  /spatial-campus-hero11/i,
  /spatial-campus-her2o/i,
  /^refs$/i
];

copyDir(path.join(ROOT_DIR, 'frontend/public'), path.join(OUT_DIR, 'frontend/public'), (fullPath, relPath, entry) => {
  if (entry.name === 'node_modules' || entry.name === '.next' || entry.name === '.git') {
    return false;
  }
  for (const pattern of excludedPublicPatterns) {
    if (pattern.test(entry.name)) {
      return false;
    }
  }
  return true;
});

// 5. Generate Production deploy.sh helper for Linux VPS
console.log('📜 Creating production deploy.sh script...');
const deployShContent = `#!/usr/bin/env bash
set -euo pipefail

echo "================================================"
echo "🚀 проект «так называемый SPARK» — Production Deploy"
echo "================================================"

if ! command -v docker >/dev/null 2>&1; then
    echo "❌ Error: Docker is not installed on this system."
    exit 1
fi

if [ ! -f .env ]; then
    echo "❌ .env not found. Copy .env.example and fill every required secret."
    echo "   The backend refuses to start with empty or default production secrets."
    exit 1
fi

if docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker-compose"
else
    echo "❌ Error: Neither 'docker compose' nor 'docker-compose' found."
    exit 1
fi

# Refuse to deploy with the placeholder secrets shipped in .env.example.
if grep -Eq '^(JWT_SECRET|ADMIN_PASSWORD|POSTGRES_PASSWORD|TELEGRAM_WEBHOOK_SECRET)=$' .env; then
    echo "❌ .env contains empty required secrets. Deployment aborted before touching running containers."
    exit 1
fi

echo "🔎 Capturing current images for rollback..."
PREVIOUS_BACKEND=$($DOCKER_COMPOSE images -q backend 2>/dev/null || true)
PREVIOUS_FRONTEND=$($DOCKER_COMPOSE images -q frontend 2>/dev/null || true)
PREVIOUS_CADDY=$($DOCKER_COMPOSE images -q caddy 2>/dev/null || true)

if [ -n "$PREVIOUS_BACKEND" ]; then docker tag "$PREVIOUS_BACKEND" spark-backend:rollback; fi
if [ -n "$PREVIOUS_FRONTEND" ]; then docker tag "$PREVIOUS_FRONTEND" spark-frontend:rollback; fi
if [ -n "$PREVIOUS_CADDY" ]; then docker tag "$PREVIOUS_CADDY" spark-caddy:rollback; fi

echo "📦 Building new images BEFORE stopping the running version..."
if ! $DOCKER_COMPOSE build backend frontend; then
    echo "❌ Build failed. The currently running version was not stopped."
    exit 1
fi

cat > docker-compose.rollback.yml <<'ROLLBACK_EOF'
services:
  backend:
    image: spark-backend:rollback
  frontend:
    image: spark-frontend:rollback
  caddy:
    image: spark-caddy:rollback
ROLLBACK_EOF

wait_for_health() {
    local container="$1"
    local deadline=$((SECONDS + 180))
    while [ $SECONDS -lt $deadline ]; do
        local status
        status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || echo missing)
        if [ "$status" = "healthy" ]; then
            return 0
        fi
        if [ "$status" = "unhealthy" ] || [ "$status" = "exited" ] || [ "$status" = "dead" ]; then
            echo "❌ Container $container is $status"
            return 1
        fi
        sleep 3
    done
    echo "❌ Timeout waiting for $container to become healthy"
    return 1
}

rollback() {
    echo "↩️ Rolling back to the previous images..."
    $DOCKER_COMPOSE down --remove-orphans >/dev/null 2>&1 || true
    if docker image inspect spark-backend:rollback >/dev/null 2>&1; then
        $DOCKER_COMPOSE -f docker-compose.yml -f docker-compose.rollback.yml up -d --no-build || true
        echo "⚠️ Rollback started. Check: $DOCKER_COMPOSE ps"
    else
        echo "❌ No previous backend image is available for rollback."
    fi
}

echo "🚀 Starting the new version..."
if ! $DOCKER_COMPOSE up -d --no-build --remove-orphans; then
    rollback
    exit 1
fi

echo "⏳ Waiting for backend readiness and frontend health..."
if ! wait_for_health spark_backend || ! wait_for_health spark_frontend || ! wait_for_health spark_caddy; then
    echo "❌ Deployment did not become healthy."
    $DOCKER_COMPOSE ps || true
    $DOCKER_COMPOSE logs --tail=80 backend || true
    rollback
    exit 1
fi

# Readiness includes PostgreSQL connectivity and migration version.
if ! $DOCKER_COMPOSE exec -T backend wget -qO- http://127.0.0.1:8080/health/ready | grep -q '"status":"ready"'; then
    echo "❌ Backend readiness probe failed after startup."
    rollback
    exit 1
fi

echo "================================================"
echo "✅ Deployment finished: all containers are healthy."
echo "🌐 Web Application: https://\${DOMAIN:-so-called-spark.ru}"
echo "📜 View live logs: $DOCKER_COMPOSE logs -f"
echo "🛑 Stop services:   $DOCKER_COMPOSE down"
echo "================================================"
`;
fs.writeFileSync(path.join(OUT_DIR, 'deploy.sh'), deployShContent.replace(/\r\n/g, '\n'), { mode: 0o755 });

// 6. Generate DEPLOY.md instruction
const deployMdContent = `# 🚀 Инструкция по деплою на сервер

Данная папка содержит полный, оптимизированный контур проекта (Frontend + Backend + PostgreSQL + Caddy). Caddy работает в режиме reverse-proxy с \`auto_https off\`: TLS обязан завершать внешний Nginx/балансировщик.

---

## Вариант 1. Запуск через архив (Самый быстрый)

1. Загрузите архив \`spark-deploy.zip\` на ваш сервер:
   \`\`\`bash
   scp spark-deploy.zip user@YOUR_SERVER_IP:/home/user/
   \`\`\`

2. Подключитесь по SSH и распакуйте:
   \`\`\`bash
   ssh user@YOUR_SERVER_IP
   mkdir -p /home/user/spark && unzip spark-deploy.zip -d /home/user/spark
   cd /home/user/spark
   \`\`\`

3. (Опционально) Отредактируйте \`.env\` при необходимости:
   \`\`\`bash
   nano .env
   \`\`\`

4. Запустите одной командой:
   \`\`\`bash
   chmod +x deploy.sh && ./deploy.sh
   \`\`\`

### Обязательная TLS-схема

Caddy в этом контуре слушает HTTP (\`auto_https off\`). Production-трафик **обязан**
приходить через внешний TLS-терминатор: Nginx, cloud load balancer или отдельный
Caddy с валидным сертификатом. Пример Nginx:

\`\`\`nginx
server {
    listen 443 ssl http2;
    server_name so-called-spark.ru;
    ssl_certificate     /etc/letsencrypt/live/so-called-spark.ru/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/so-called-spark.ru/privkey.pem;
    location / { proxy_pass http://127.0.0.1:38080; proxy_set_header Host $host; }
}
\`\`\`

Без внешнего терминатора сайт будет доступен только по HTTP, а Telegram и
платёжные webhook-и должны быть настроены на HTTPS-домен.

### Проверка и откат

\`deploy.sh\` собирает образы до остановки текущей версии, ждёт health-статусы
backend/frontend/caddy и проверяет \`/health/ready\` (PostgreSQL + версия миграций).
При любой ошибке он переключается на сохранённые образы \`spark-*-:rollback\`.

### Восстановление backup

\`\`\`bash
gunzip -c /backups/spark_db_YYYYMMDD_HHMMSS.sql.gz > /tmp/restore.sql
createdb -h 127.0.0.1 -U spark_user spark_restore_test
psql -h 127.0.0.1 -U spark_user -d spark_restore_test -f /tmp/restore.sql
\`\`\`

Восстановление всегда выполняйте в отдельную базу и проверяйте её перед
переключением production.

---

## Вариант 2. Запуск через папку / rsync

1. Синхронизируйте папку \`deploy/\` с сервером:
   \`\`\`bash
   rsync -avz --progress ./deploy/ user@YOUR_SERVER_IP:/home/user/spark/
   \`\`\`

2. На сервере:
   \`\`\`bash
   cd /home/user/spark
   chmod +x deploy.sh && ./deploy.sh
   \`\`\`

---

## Что внутри контура:
* **caddy** (порт 38080/80): Reverse-proxy, маршрутизация \`/api/*\` на Go, фронтенда на Next.js.
* **frontend** (Next.js 14): Оптимизированный продакшн-бандл.
* **backend** (Golang 1.22): REST API, JWT-авторизация, админ-панель и встроенный Telegram-бот на чистом Go.
* **postgres** (PostgreSQL 16 Alpine): База данных с авто-накатом \`seed.sql\`.
`;
fs.writeFileSync(path.join(OUT_DIR, 'DEPLOY.md'), deployMdContent, 'utf8');

// 7. Calculate package statistics
function getDirStats(dir) {
  let count = 0;
  let size = 0;
  const list = fs.readdirSync(dir, { withFileTypes: true });
  for (const item of list) {
    const full = path.join(dir, item.name);
    if (item.isDirectory()) {
      const sub = getDirStats(full);
      count += sub.count;
      size += sub.size;
    } else {
      count++;
      size += fs.statSync(full).size;
    }
  }
  return { count, size };
}

const stats = getDirStats(OUT_DIR);
const sizeMb = (stats.size / (1024 * 1024)).toFixed(2);

console.log('📁 Deploy folder created at:', OUT_DIR);
console.log(`📊 Total files: ${stats.count} | Size: ${sizeMb} MB`);

// 8. Create ZIP archive
console.log('🗜️ Creating spark-deploy.zip archive...');
try {
  if (process.platform === 'win32') {
    cp.execSync(`powershell -NoProfile -Command "Compress-Archive -Path '${OUT_DIR}\\*' -DestinationPath '${ZIP_PATH}' -Force"`, { stdio: 'inherit' });
  } else {
    cp.execSync(`cd "${OUT_DIR}" && zip -r "${ZIP_PATH}" .`, { stdio: 'inherit' });
  }
  if (fs.existsSync(ZIP_PATH)) {
    const zipSize = (fs.statSync(ZIP_PATH).size / (1024 * 1024)).toFixed(2);
    console.log(`✅ Archive created: ${ZIP_PATH} (${zipSize} MB)`);
  }
} catch (err) {
  console.warn('⚠️ Could not automatically create zip archive, but deploy folder is 100% ready:', err.message);
}

console.log('\n🎉 Done! Ready to upload to server.');
