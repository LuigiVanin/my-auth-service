# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies if needed (e.g. git for private repos, though not needed here)
# RUN apk add --no-cache git

# Copy dependency definitions
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# CGO_ENABLED=0 creates a statically linked binary, which is better for alpine/scratch
RUN CGO_ENABLED=0 GOOS=linux go build -o auth_service ./cmd/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/auth_service .

# Copy production config if it exists
# This command will not fail if the file doesn't exist, but it's better to be explicit
COPY .env.prod.yaml .

# Create a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Expose the default port
EXPOSE 3333

# Set default environment variables
ENV SERVER_PORT=3333
ENV APP_ENV=prod

# Run the application
CMD ["./auth_service", "prod"]
