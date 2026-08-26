# Stage 1: Build stage
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=""

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
    -o engine-server main.go

# Stage 2: Minimal Production Image
FROM alpine:3.21

LABEL org.opencontainers.image.title="triage-engine" \
      org.opencontainers.image.description="Zero-overhead Go panic isolation, AST slicing & Multi-Provider AI triage engine" \
      org.opencontainers.image.source="https://github.com/algotyrnt/triage" \
      org.opencontainers.image.licenses="Apache-2.0"

RUN apk --no-cache add ca-certificates tzdata wget

RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/engine-server /app/engine-server

USER appuser:appgroup

ENV PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/engine-server"]
