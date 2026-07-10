# syntax=docker/dockerfile:1
# Build the static Next.js export.
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
# Cache npm's download + install cache between builds.
RUN --mount=type=cache,target=/root/.npm \
    npm ci
COPY . .
RUN npm run build

# Serve the static export with nginx, proxying /api to the torrent-service.
FROM nginx:alpine
RUN apk add --no-cache wget
COPY --from=builder /app/out /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
