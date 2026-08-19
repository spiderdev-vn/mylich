# ===================================================
# Dockerfile for lich-server (Self-hosted Calendar)
# Base: Node.js 22 Alpine (Lightweight & Native TS)
# ===================================================

FROM node:22-alpine AS base

# Install curl for healthcheck
RUN apk add --no-cache curl

WORKDIR /app

# Copy dependency definitions
COPY package.json package-lock.json ./

# Install production dependencies only
RUN npm ci --omit=dev && npm cache clean --force

# Copy application source code and migrations
COPY tsconfig.json ./
COPY src/ ./src/

# Create persistent data directory and set permissions for non-root user
RUN mkdir -p /data && chown -R node:node /data /app

# Switch to non-root user
USER node

# Environment defaults
ENV NODE_ENV=production \
    HOST=0.0.0.0 \
    PORT=3000 \
    DATABASE_PATH=/data/lich.db \
    LOG_LEVEL=info

# Expose HTTP port
EXPOSE 3000

# Persistent volume for SQLite database
VOLUME ["/data"]

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://127.0.0.1:3000/health || exit 1

# Start server
CMD ["node", "src/server.ts"]
