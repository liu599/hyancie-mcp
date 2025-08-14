# Build stage
FROM harbor01.litcompute.com/dev-center-public/golang:1.23 as builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
# RUN go mod download
RUN go env -w GOPROXY=https://goproxy.cn,direct

# 下载依赖
RUN [ ! -d "vendor" ] && go mod download all || echo "skipping..."

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o hyancie-mcp ./cmd/hyancie

# Final stage
FROM harbor01.litcompute.com/dev-center/alpine:3.20.2 as prod

# Install ca-certificates for HTTPS requests
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN useradd -r -u 1000 -m hyancie

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder --chown=1000:1000 /app/hyancie-mcp /app/

# Copy the configuration file
COPY --chown=1000:1000 config.sample.json /app/config.json

# Use the non-root user
USER hyancie

# Expose the port the app runs on
EXPOSE 8001

# Run the application
ENTRYPOINT ["/app/hyancie-mcp", "--transport", "sse", "--sse-address", "0.0.0.0:8001"]