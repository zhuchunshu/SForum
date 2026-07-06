FROM golang:1.26-alpine AS base

WORKDIR /app/apps/api
RUN apk add --no-cache ca-certificates git

COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download

COPY apps/api/ ./
COPY extensions/builtin /app/extensions/builtin

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
RUN bun install --frozen-lockfile
COPY apps/web/ ./
COPY extensions/builtin /app/extensions/builtin

FROM alpine:3.22 AS api

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/sforum-api /usr/local/bin/sforum-api
COPY --from=build /app/extensions/builtin /app/extensions/builtin
EXPOSE 8080
CMD ["sforum-api"]

FROM oven/bun:1.3-alpine AS worker

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/sforum-worker /usr/local/bin/sforum-worker
COPY --from=build /app/extensions/builtin /app/extensions/builtin
COPY --from=webdeps /app/apps/web /app/apps/web
ENV THEME_WEB_ROOT=/app/apps/web
ENV THEME_BUN_PATH=bun
ENV THEME_RELEASE_ROOT=/var/lib/sforum/theme-releases
CMD ["sforum-worker"]

FROM alpine:3.22 AS migrate

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/sforum-migrate /usr/local/bin/sforum-migrate
CMD ["sforum-migrate"]
