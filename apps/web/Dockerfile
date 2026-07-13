FROM oven/bun:1.3-alpine AS deps

WORKDIR /app/apps/web
COPY apps/web/package.json apps/web/bun.lock ./
COPY apps/web/packages/admin-sdk/package.json ./packages/admin-sdk/package.json
RUN bun install --frozen-lockfile

FROM deps AS dev

COPY apps/web/ ./
COPY extensions/builtin /app/extensions/builtin
EXPOSE 3000
CMD ["bun", "run", "dev"]

FROM deps AS build

COPY apps/web/ ./
COPY extensions/builtin /app/extensions/builtin
RUN ln -sfn .nuxt-build .nuxt && bun run build

FROM alpine:3.22 AS prod

# H6：容器以非 root 用户运行。
RUN apk add --no-cache nodejs && \
	addgroup -S sforum && adduser -S -G sforum sforum && \
	mkdir -p /app/apps/web && \
	chown -R sforum:sforum /app
WORKDIR /app/apps/web
ENV HOST=0.0.0.0
ENV PORT=3000
COPY --from=build --chown=sforum:sforum /app/apps/web/.output ./.output
USER sforum
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
