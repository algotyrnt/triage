# Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
# SPDX-License-Identifier: Apache-2.0

# Stage 1: Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy root monorepo files
COPY go.mod go.sum ./
COPY apps/engine ./apps/engine

WORKDIR /app/apps/engine
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o engine-server main.go

# Stage 2: Minimal Production Image
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/apps/engine/engine-server /app/engine-server

USER appuser:appgroup

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/engine-server"]
