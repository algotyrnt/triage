# Stage 1: Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy root monorepo files
COPY go.mod go.sum ./
COPY sdk/go ./sdk/go
COPY test-service ./test-service

WORKDIR /app/test-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o test-service main.go

# Stage 2: Minimal Production Image
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/test-service/test-service /app/test-service

USER appuser:appgroup

ENV PORT=8081
EXPOSE 8081

ENTRYPOINT ["/app/test-service"]
