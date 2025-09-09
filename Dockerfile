# Multi-stage build for ChoreoAtlas CLI
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go modules files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X 'github.com/choreoatlas2025/cli/internal/cli.Version=docker' -X 'github.com/choreoatlas2025/cli/internal/cli.BuildEdition=ce'" \
    -o choreoatlas ./cmd/choreoatlas

# Final stage using distroless
FROM gcr.io/distroless/static:nonroot

# Copy ca-certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/choreoatlas /usr/local/bin/choreoatlas

# Copy examples for demonstration
COPY --from=builder /app/examples /examples

# Use non-root user
USER nonroot:nonroot

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/choreoatlas"]

# Default command
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="ChoreoAtlas CLI"
LABEL org.opencontainers.image.description="从真实追踪中映射-验证-引导跨服务编排的契约即代码工具"
LABEL org.opencontainers.image.source="https://github.com/choreoatlas2025/cli"
LABEL org.opencontainers.image.url="https://choreoatlas.io"
LABEL org.opencontainers.image.documentation="https://choreoatlas.io"
LABEL org.opencontainers.image.vendor="ChoreoAtlas"
LABEL org.opencontainers.image.licenses="Apache-2.0"