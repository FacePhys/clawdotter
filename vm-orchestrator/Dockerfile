# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /orchestrator ./cmd/orchestrator

# Production stage
FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    iptables \
    iproute2 \
    bridge-utils \
    curl

COPY --from=builder /orchestrator /usr/local/bin/orchestrator

# Firecracker directories
RUN mkdir -p /var/lib/firecracker/rootfs \
    /var/lib/firecracker/vms \
    /tmp/firecracker/sockets

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/orchestrator"]
