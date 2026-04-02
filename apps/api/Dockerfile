FROM golang:1.24-alpine AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /src

# Copy workspace root and all modules together so go.work resolution works
COPY go.work go.work.sum ./
COPY apps/ ./apps/
COPY packages/ ./packages/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /api ./apps/api/cmd/server

# --- runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /api /api
COPY --from=builder /src/apps/api/migrations /migrations

ENV MIGRATIONS_PATH=/migrations
EXPOSE 8080
ENTRYPOINT ["/api"]
