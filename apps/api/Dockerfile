FROM golang:1.26.5-alpine AS base

WORKDIR /app/apps/api
RUN apk add --no-cache ca-certificates git

COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download

COPY apps/api/ ./
COPY extensions/builtin /app/extensions/builtin
# 内置插件后端二进制（与 scripts/build-builtin-plugins.sh 对齐）
RUN cd /app/extensions/builtin/plugins/sforum-smtp/backend && \
	CGO_ENABLED=0 go build -o plugin . && \
	cd /app/extensions/builtin/plugins/sforum-content-policy/backend && \
	CGO_ENABLED=0 go build -o plugin .

FROM base AS dev

RUN go install github.com/air-verse/air@latest
CMD ["air", "-c", ".air.toml"]

FROM base AS build

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sforum-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sforum-worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sforum-migrate ./cmd/migrate

FROM oven/bun:1.3-alpine AS webdeps

WORKDIR /app/apps/web
COPY apps/web/package.json apps/web/bun.lock ./
COPY apps/web/packages/admin-sdk/package.json ./packages/admin-sdk/package.json
RUN bun install --frozen-lockfile
COPY apps/web/ ./
COPY extensions/builtin /app/extensions/builtin

FROM alpine:3.22 AS websource

WORKDIR /app/apps/web
COPY apps/web/ ./

FROM alpine:3.22 AS api

RUN apk add --no-cache ca-certificates && \
	# H6：容器以非 root 用户运行，RCE/路径穿越后权限受限。
	addgroup -S sforum && adduser -S -G sforum sforum && \
	mkdir -p /app/storage/app/attachments /var/lib/sforum/extensions /var/lib/sforum/theme-releases && \
	chown -R sforum:sforum /app /var/lib/sforum
WORKDIR /app
COPY --from=build --chown=sforum:sforum /out/sforum-api /usr/local/bin/sforum-api
COPY --from=build --chown=sforum:sforum /app/extensions/builtin /app/extensions/builtin
COPY --from=websource --chown=sforum:sforum /app/apps/web /app/apps/web
ENV WEB_RELEASE_WEB_ROOT=/app/apps/web
USER sforum
EXPOSE 8080
CMD ["sforum-api"]

FROM oven/bun:1.3-alpine AS worker

RUN apk add --no-cache ca-certificates nodejs && \
	addgroup -S sforum && adduser -S -G sforum sforum && \
	mkdir -p /app/storage/app/attachments /var/lib/sforum/extensions /var/lib/sforum/theme-releases /app/apps/web && \
	chown -R sforum:sforum /app /var/lib/sforum
WORKDIR /app
COPY --from=build --chown=sforum:sforum /out/sforum-worker /usr/local/bin/sforum-worker
COPY --from=build --chown=sforum:sforum /app/extensions/builtin /app/extensions/builtin
COPY --from=webdeps --chown=sforum:sforum /app/apps/web /app/apps/web
ENV THEME_WEB_ROOT=/app/apps/web
ENV THEME_BUN_PATH=bun
ENV THEME_RELEASE_ROOT=/var/lib/sforum/theme-releases
USER sforum
CMD ["sforum-worker"]

FROM alpine:3.22 AS migrate

RUN apk add --no-cache ca-certificates && \
	addgroup -S sforum && adduser -S -G sforum sforum
WORKDIR /app
COPY --from=build /out/sforum-migrate /usr/local/bin/sforum-migrate
USER sforum
CMD ["sforum-migrate"]
