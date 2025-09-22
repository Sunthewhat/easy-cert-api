# Build stage
FROM golang:latest AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage with Node.js and Bun support
FROM node:18-alpine AS runner

# Install system dependencies for canvas and other native modules
RUN apk --no-cache add \
    ca-certificates \
    cairo-dev \
    pango-dev \
    jpeg-dev \
    giflib-dev \
    librsvg-dev \
    pixman-dev \
    pangomm-dev \
    libjpeg-turbo-dev \
    freetype-dev \
    python3 \
    make \
    g++ \
    curl \
    unzip

# Install Bun
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

# Verify Bun installation
RUN bun --version

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy config file if needed
COPY --from=builder /app/config.yml ./config.yml

# Copy internal renderer assets for embedded renderer
COPY --from=builder /app/internal/renderer/package.json ./internal/renderer/package.json
COPY --from=builder /app/internal/renderer/renderer.ts ./internal/renderer/renderer.ts

# Pre-install Bun dependencies for embedded renderer to speed up runtime
WORKDIR /root/internal/renderer
RUN bun install

# Return to main working directory
WORKDIR /root/

# Expose port
EXPOSE 8000

# Run the binary
ENTRYPOINT ["./main"]