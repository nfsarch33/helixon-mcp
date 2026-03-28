# syntax=docker/dockerfile:1

# ──────────────────────────────────────────
# Stage 1: Build
# ──────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

# Cache dependencies layer separately
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w" -o /bin/ironclaw-mcp ./cmd/ironclaw-mcp

# ──────────────────────────────────────────
# Stage 2: Minimal runtime image
# ──────────────────────────────────────────
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/ironclaw-mcp /ironclaw-mcp

# stdio MCP server — no ports needed for stdio transport
# For SSE transport expose 8080
EXPOSE 8080

ENV MCP_TRANSPORT=stdio \
    IRONCLAW_BASE_URL=http://localhost:3000 \
    LOG_LEVEL=info

ENTRYPOINT ["/ironclaw-mcp"]
