# VerustCode Dockerfile
# Multi-stage build: Go + Node builder -> Alpine runtime

# =============================================================================
# Stage 1: Builder
# Contains Go 1.24 and Node 22 for building frontend and backend
# =============================================================================
FROM golang:1.24-alpine AS builder

# Install build dependencies
# - nodejs/npm: Frontend build
# - make/git: Build tools
# - ca-certificates: HTTPS support
RUN apk add --no-cache \
    nodejs \
    npm \
    make \
    git \
    ca-certificates

# Set working directory
WORKDIR /build

# Copy dependency files first for better cache utilization
COPY go.mod go.sum ./
COPY frontend/package.json frontend/package-lock.json ./frontend/

# Download Go dependencies
RUN go mod download

# Install frontend dependencies
RUN cd frontend && npm ci

# Copy source code
COPY . .

# Build frontend and backend using Makefile
# This ensures consistency with local development builds
ARG VERSION=docker
ARG BUILD_TIME
ARG GIT_COMMIT
RUN make build-all VERSION=${VERSION}

# =============================================================================
# Stage 2: Runtime
# Minimal Alpine image with only the compiled binary
# =============================================================================
FROM alpine:3.21

# Install runtime dependencies
# - ca-certificates: HTTPS support for API calls
# - tzdata: Timezone support
# - git: Required for cloning repositories during review/report tasks
# - chromium: Headless browser for PDF export (chromedp)
# - font-noto-cjk: CJK fonts for Chinese/Japanese/Korean support in PDF
# - font-noto: Base Noto fonts for better international text rendering
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    git \
    chromium \
    font-noto-cjk \
    font-noto

# Set Chrome path environment variable for chromedp
ENV CHROME_PATH=/usr/bin/chromium-browser

# Create non-root user for security
RUN addgroup -g 1000 verustcode && \
    adduser -u 1000 -G verustcode -s /bin/sh -D verustcode

# Set working directory
WORKDIR /app

# Create necessary directories with proper permissions
RUN mkdir -p /app/config /app/data /app/logs /app/workspace /app/report_workspace /app/review_results && \
    chown -R verustcode:verustcode /app

# Copy binary from builder stage
COPY --from=builder /build/verustcode /app/verustcode

# Copy default config files (will be overwritten by volume mounts)
COPY --from=builder /build/config/bootstrap.example.yaml /app/config/bootstrap.example.yaml
COPY --from=builder /build/config/reviews/default.example.yaml /app/config/reviews/default.example.yaml

# Set ownership
RUN chown -R verustcode:verustcode /app

# Switch to non-root user
USER verustcode

# Expose ports
# 8091: Main HTTP server (Web UI / API / Webhooks)
# 9090: Prometheus metrics endpoint (when telemetry enabled)
EXPOSE 8091 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8091/api/health || exit 1

# Default command
ENTRYPOINT ["/app/verustcode"]
CMD ["serve"]


