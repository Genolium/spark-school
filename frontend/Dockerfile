# Stage 1: Install dependencies
FROM node:22-alpine AS deps
RUN apk add --no-cache libc6-compat
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

# Stage 2: Build Next.js application
FROM node:22-alpine AS builder
WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

ARG NEXT_PUBLIC_API_URL=https://sparkprep.ru/api/v1
ARG NEXT_PUBLIC_BOT_USERNAME=spark_prep_bot
ARG NEXT_PUBLIC_KINESCOPE_LESSON1_ID=
ARG NEXT_PUBLIC_KINESCOPE_LESSON2_ID=
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_BOT_USERNAME=$NEXT_PUBLIC_BOT_USERNAME
ENV NEXT_PUBLIC_KINESCOPE_LESSON1_ID=$NEXT_PUBLIC_KINESCOPE_LESSON1_ID
ENV NEXT_PUBLIC_KINESCOPE_LESSON2_ID=$NEXT_PUBLIC_KINESCOPE_LESSON2_ID

ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production

RUN npm run build

# Stage 3: Production-only dependencies (no tailwind/eslint/dev tooling)
FROM node:22-alpine AS prod-deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev

# Stage 4: Runner
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder --chown=nextjs:nodejs /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next ./.next
COPY --from=prod-deps --chown=nextjs:nodejs /app/node_modules ./node_modules
COPY --from=builder --chown=nextjs:nodejs /app/package.json ./package.json

# The runtime only needs Next.js. npm bundles vulnerable packages
# (brace-expansion, pacote, sigstore) that are not used to serve the app.
RUN rm -rf /usr/local/lib/node_modules/npm /usr/local/bin/npm /usr/local/bin/npx \
    && chmod -R a+rX /app/public /app/.next

USER nextjs

EXPOSE 3000

CMD ["./node_modules/.bin/next", "start", "-p", "3000"]
