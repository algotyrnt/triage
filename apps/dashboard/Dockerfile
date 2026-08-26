FROM oven/bun:alpine AS base
WORKDIR /app

FROM base AS dependencies
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile

FROM base AS builder
ARG APP_VERSION=dev
COPY --from=dependencies /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN bun run build

FROM base AS runner
WORKDIR /app

LABEL org.opencontainers.image.title="triage-dashboard" \
      org.opencontainers.image.description="Next.js 16 Studio Dashboard for Go Crash Isolation & AST Triage" \
      org.opencontainers.image.source="https://github.com/algotyrnt/triage" \
      org.opencontainers.image.licenses="Apache-2.0"

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN apk --no-cache add wget

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder --chown=nextjs:nodejs /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:3000 || exit 1

CMD ["bun", "server.js"]
