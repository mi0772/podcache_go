# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go module files
COPY go.mod* go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o podcache .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S podcache && \
    adduser -u 1001 -S podcache -G podcache

WORKDIR /home/podcache

# Copy the binary from builder stage
COPY --from=builder /app/podcache .

# Change ownership
RUN chown podcache:podcache /home/podcache/podcache

# Switch to non-root user
USER podcache

# Expose port
EXPOSE 6379

# Create volume for cache data
VOLUME ["/home/podcache/.cas"]

# Run the binary
CMD ["./podcache"]
