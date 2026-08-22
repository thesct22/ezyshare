# syntax=docker/dockerfile:1

# ---------------------------------------------
# Builder Stage
# ---------------------------------------------
FROM node:22-alpine AS builder
WORKDIR /app

# Install dependencies
COPY package.json package-lock.json* ./
RUN npm ci || npm install

# Copy source and build
COPY . .
RUN npm run build

# ---------------------------------------------
# Development Target
# ---------------------------------------------
FROM node:22-alpine AS dev
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci || npm install
COPY . .
# Start Vite with --host to allow external connections
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

# ---------------------------------------------
# Production Target (Hardened Nginx Unprivileged)
# ---------------------------------------------
FROM nginxinc/nginx-unprivileged:alpine AS prod

LABEL org.opencontainers.image.title="EzyShare Frontend" \
      org.opencontainers.image.description="React & MUI Web App for EzyShare Zero-Knowledge File Sharing" \
      org.opencontainers.image.source="https://github.com/thesct22/ezyshare" \
      org.opencontainers.image.url="https://github.com/thesct22/ezyshare" \
      org.opencontainers.image.licenses="MIT"

# Switch to root to configure the container
USER root

# Remove default nginx static assets
RUN rm -rf /usr/share/nginx/html/*

# Copy built assets from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy custom Nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Switch back to non-root user
USER 101

# Port 8080 is used by unprivileged nginx
EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
