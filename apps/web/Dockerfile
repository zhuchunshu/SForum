FROM oven/bun:1.3-alpine AS deps

WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

FROM deps AS dev

COPY . .
EXPOSE 3000
CMD ["bun", "run", "dev"]

FROM deps AS build

COPY . .
RUN bun run build

FROM oven/bun:1.3-alpine AS prod

WORKDIR /app
ENV HOST=0.0.0.0
ENV PORT=3000
COPY --from=build /app/.output ./.output
EXPOSE 3000
CMD ["bun", ".output/server/index.mjs"]
