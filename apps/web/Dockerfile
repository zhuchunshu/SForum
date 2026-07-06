FROM oven/bun:1.3-alpine AS deps

WORKDIR /app/apps/web
COPY apps/web/package.json apps/web/bun.lock ./
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

FROM oven/bun:1.3-alpine AS prod

WORKDIR /app
ENV HOST=0.0.0.0
ENV PORT=3000
COPY --from=build /app/apps/web/.output ./.output
EXPOSE 3000
CMD ["bun", ".output/server/index.mjs"]
