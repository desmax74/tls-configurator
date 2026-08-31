# Build stage
FROM registry.access.redhat.com/hi/go:1.26.7 AS builder

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /tls-configurator \
    ./cmd/tls-configurator

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

# Update and install runtime dependencies
RUN microdnf update -y && \
    microdnf install -y ca-certificates && \
    microdnf clean all

# Create non-root user
RUN groupadd -g 1000 app && \
    useradd -u 1000 -g app -s /bin/bash -m app

# Copy binary from builder
COPY --from=builder /tls-configurator /usr/local/bin/tls-configurator

# Set ownership
RUN chown app:app /usr/local/bin/tls-configurator

# Switch to non-root user
USER app

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/tls-configurator"]

# Default command
CMD ["--help"]
