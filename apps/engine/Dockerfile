# ==============================================================================
# Multi-Stage Production Dockerfile for Triage Telemetry Engine
# ==============================================================================

# STAGE 1: Build binary
FROM golang:alpine AS builder

WORKDIR /app

# Install ca-certificates for TLS HTTPS API calls
RUN apk add --no-cache ca-certificates git

# Copy module definitions and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build static binary without CGO
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/engine main.go

# STAGE 2: Lightweight production runner
FROM alpine:3.20 AS runner

WORKDIR /app

# Add ca-certificates and create unprivileged user
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

# Copy compiled binary from builder
COPY --from=builder /app/engine /app/engine

# Change ownership to non-root user
USER appuser:appgroup

# Environment defaults
ENV PORT=8080
ENV AST_WORKSPACE_ROOT=/app

EXPOSE 8080

# Health check probe
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/engine"]
