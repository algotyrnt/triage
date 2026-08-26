FROM oven/bun:alpine AS builder
WORKDIR /app

COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile

COPY . .
RUN bun run build

FROM nginx:alpine AS runner

LABEL org.opencontainers.image.title="triage-web" \
      org.opencontainers.image.description="Public documentation and website for Triage" \
      org.opencontainers.image.source="https://github.com/algotyrnt/triage" \
      org.opencontainers.image.licenses="Apache-2.0"

COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:80 || exit 1

CMD ["nginx", "-g", "daemon off;"]
