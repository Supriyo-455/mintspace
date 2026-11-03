# ==============================================================================
# MintSpace Production Multi-Stage Dockerfile
# Compact, hardened, non-root container deployment (< 25MB final image)
# ==============================================================================

# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY . .

# Compile optimized static binary with embedded symbol tables stripped
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/mintspace .

# Runtime Hardened Stage
FROM alpine:3.20

WORKDIR /app

# Install runtime certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -S mintspace && adduser -S mintspace -G mintspace

# Create application data and log directories
RUN mkdir -p /app/data/blobs /app/logs /app/static /app/templates && \
    chown -R mintspace:mintspace /app

# Copy binary and assets from builder
COPY --from=builder /app/mintspace /app/mintspace
COPY --from=builder /app/static /app/static
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/config.json /app/config.json

# Switch to non-root user
USER mintspace

# Environment defaults
ENV PORT=3000
EXPOSE 3000

# Health check for orchestrators
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://127.0.0.1:3000/healthz || exit 1

ENTRYPOINT ["/app/mintspace"]
