FROM node:20-alpine AS builder
WORKDIR /app

COPY apps/web/package*.json ./
RUN npm ci

COPY apps/web/ .

# Baked into the client bundle at build time
ARG NUXT_PUBLIC_API_BASE=https://api.h2oflows.app
ENV NUXT_PUBLIC_API_BASE=$NUXT_PUBLIC_API_BASE

RUN npm run build

# --- runtime ---
FROM node:20-alpine
WORKDIR /app

COPY --from=builder /app/.output ./.output

ENV NODE_ENV=production
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
