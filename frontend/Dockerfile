FROM node:24.13.0-alpine AS build
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build
ARG APP_REVISION=local
RUN printf '{"revision":"%s"}\n' "$APP_REVISION" > dist/release.json
FROM nginx:1.28.0-alpine
COPY infra/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
