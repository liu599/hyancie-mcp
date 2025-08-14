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
RUN go build -o hyancie-mcp ./cmd/hyancie

# Final stage
FROM debian:bullseye-slim

# Install ca-certificates for HTTPS requests
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN useradd -r -u 1000 -m hyancie

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder --chown=1000:1000 /app/hyancie-mcp /app/

# Copy the configuration file
COPY --chown=1000:1000 config.json /app/

# Use the non-root user
USER hyancie

# Expose the port the app runs on
EXPOSE 8001

# Run the application
ENTRYPOINT ["/app/hyancie-mcp", "--transport", "sse", "--sse-address", "0.0.0.0:8001"]