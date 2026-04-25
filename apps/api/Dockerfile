FROM golang:1.25-alpine AS builder
WORKDIR /src

# Copy workspace root and all modules together so go.work resolution works
COPY go.work go.work.sum ./
COPY apps/ ./apps/
COPY packages/ ./packages/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /api             ./apps/api/cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /seed-reaches    ./apps/api/cmd/seed-reaches && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /seed-descs      ./apps/api/cmd/seed-reach-descriptions && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /seed-flows      ./apps/api/cmd/seed-flow-ranges && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /import-kml      ./apps/api/cmd/import-kml && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /embed-reaches   ./apps/api/cmd/embed-reaches && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /backfill-comids ./apps/api/cmd/backfill-comids

# --- runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /api             /api
COPY --from=builder /seed-reaches    /seed-reaches
COPY --from=builder /seed-descs      /seed-descs
COPY --from=builder /seed-flows      /seed-flows
COPY --from=builder /import-kml      /import-kml
COPY --from=builder /embed-reaches   /embed-reaches
COPY --from=builder /backfill-comids /backfill-comids
COPY --from=builder /src/apps/api/migrations /migrations

ENV MIGRATIONS_PATH=/migrations
EXPOSE 8080
ENTRYPOINT ["/api"]
