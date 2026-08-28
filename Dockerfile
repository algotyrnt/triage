# Stage 1: Build the Vite Studio Dashboard
FROM oven/bun:alpine AS dashboard-builder
WORKDIR /build/dashboard
COPY dashboard/package.json dashboard/bun.lock* ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile
COPY dashboard .
RUN bun run build

# Stage 2: Build the Go Triage binary with embedded UI
FROM golang:alpine AS server-builder
WORKDIR /build/engine
COPY engine/go.mod engine/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY engine .
COPY --from=dashboard-builder /build/engine/internal/ui/dist ./internal/ui/dist

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=""

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
    -o /build/triage main.go

# Stage 3: Minimal Production Image (Zero-Dependency Single Container)
FROM alpine:3.21

LABEL org.opencontainers.image.title="triage" \
      org.opencontainers.image.description="Zero-overhead Go panic isolation, AST slicing & AI triage server with embedded studio dashboard" \
      org.opencontainers.image.source="https://github.com/algotyrnt/triage" \
      org.opencontainers.image.licenses="Apache-2.0"

RUN apk --no-cache add ca-certificates tzdata wget

RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app
RUN mkdir -p /data && chown -R appuser:appgroup /data /app

COPY --from=server-builder --chown=appuser:appgroup /build/triage /app/triage

USER appuser:appgroup

VOLUME ["/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/triage"]
CMD ["--data-dir=/data", "--port=8080"]
