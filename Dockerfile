FROM node:24-alpine AS web-build
WORKDIR /src/apps/web
COPY apps/web/package.json ./package.json
COPY apps/web/ .
RUN npm install && npm run build

FROM node:24-alpine AS admin-build
WORKDIR /src/apps/admin
COPY apps/admin/package.json ./package.json
COPY apps/admin/ .
RUN npm install && npm run build

FROM golang:1.26-alpine AS api-build
WORKDIR /src
COPY apps/api/go.mod ./apps/api/
RUN cd apps/api && go mod download
COPY apps/api ./apps/api
RUN cd apps/api && go build -o /out/nusamedia ./cmd/server

FROM alpine:3.21
RUN addgroup -S nusamedia && adduser -S nusamedia -G nusamedia
COPY --from=api-build /out/nusamedia /usr/local/bin/nusamedia
COPY --from=web-build /src/apps/web/dist /opt/nusamedia/web
COPY --from=admin-build /src/apps/admin/dist /opt/nusamedia/admin
ENV WEB_DIR=/opt/nusamedia/web ADMIN_DIR=/opt/nusamedia/admin PORT=8080
EXPOSE 8080
USER nusamedia
ENTRYPOINT ["/usr/local/bin/nusamedia"]
