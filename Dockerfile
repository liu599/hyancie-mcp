# Build stage
FROM golang:1.24-bullseye AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o inori-mcp ./cmd/inori-mcp

# Final stage
FROM debian:bullseye-slim

# Install ca-certificates for HTTPS requests
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN useradd -r -u 1000 -m inori-mcp

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder --chown=1000:1000 /app/inori-mcp /app/

# Use the non-root user
USER inori-mcp

# Expose the port the app runs on
EXPOSE 8000

# Run the application
ENTRYPOINT ["/app/inori-mcp", "--transport", "sse", "--sse-address", "0.0.0.0:8000"]