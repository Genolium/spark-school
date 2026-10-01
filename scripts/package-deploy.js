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
  'MAIN.MD'
];
rootFiles.forEach(f => copyFileSafe(f));

// (Do not copy .env to deploy folder - server keeps its own .env or uses .env.example)

// Copy n8n-workflows if exists
if (fs.existsSync(path.join(ROOT_DIR, 'n8n-workflows'))) {
  console.log('🤖 Copying n8n workflows templates...');
  copyDir(path.join(ROOT_DIR, 'n8n-workflows'), path.join(OUT_DIR, 'n8n-workflows'));
}

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
  'frontend/pnpm-lock.yaml',
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
set -e

echo "================================================"
echo "🚀 проект «так называемый SPARK» — Production Deploy"
echo "================================================"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed on this system."
    echo "💡 Please install Docker: https://docs.docker.com/engine/install/"
    exit 1
fi

# Ensure .env file exists
if [ ! -f .env ]; then
    echo "📋 .env not found. Copying from .env.example..."
    cp .env.example .env
fi

# Detect Docker Compose command
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    echo "❌ Error: Neither 'docker compose' nor 'docker-compose' found."
    exit 1
fi

echo "📦 Building and launching containers..."
$DOCKER_COMPOSE down --remove-orphans 2>/dev/null || true
docker rm -f spark_caddy spark_backend spark_frontend spark_postgres spark_n8n spark_bot 2>/dev/null || true
$DOCKER_COMPOSE up -d --build

echo "⏳ Checking container health..."
sleep 5
$DOCKER_COMPOSE ps

echo "================================================"
echo "✅ Deployment finished successfully!"
echo "🌐 Web Application: http://localhost or https://\${DOMAIN:-so-called-spark.ru}"
echo "📜 View live logs: $DOCKER_COMPOSE logs -f"
echo "🛑 Stop services:   $DOCKER_COMPOSE down"
echo "================================================"
`;
fs.writeFileSync(path.join(OUT_DIR, 'deploy.sh'), deployShContent.replace(/\r\n/g, '\n'), { mode: 0o755 });

// 6. Generate DEPLOY.md instruction
const deployMdContent = `# 🚀 Инструкция по деплою на сервер

Данная папка содержит полный, оптимизированный контур проекта (Frontend + Backend + PostgreSQL + Caddy SSL) без мусора и лишних зависимостей.

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
* **backend** (Golang 1.22): REST API, JWT-авторизация, админ-панель.
* **n8n** (порт 38567 или \`https://n8n.so-called-spark.ru\`): Визуальный конструктор воронок и диалогов Telegram-бота. Готовый шаблон для импорта лежит в папке \`n8n-workflows/spark-bot-workflow.json\`.
* **postgres** (PostgreSQL 16 Alpine): База данных с авто-накатом \`seed.sql\`.

---

## 🤖 Как настроить Telegram-бота в n8n:
1. Откройте в браузере \`https://n8n.YOUR_DOMAIN\` (или \`http://YOUR_SERVER_IP:5678\`).
2. При первом входе создайте аккаунт администратора n8n.
3. Нажмите **Add Workflow** ➔ в меню справа выберите **Import from File** и укажите \`n8n-workflows/spark-bot-workflow.json\`.
4. В ноде **Telegram Trigger** добавьте свои учетные данные бота (токен от @BotFather).
5. Нажмите кнопку **Save** и включите переключатель **Active**. Бот сразу начинает обрабатывать сообщения!
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
