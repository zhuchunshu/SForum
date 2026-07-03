FROM golang:1.26-alpine AS base

WORKDIR /app
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

FROM base AS dev

RUN go install github.com/air-verse/air@latest
CMD ["air", "-c", ".air.toml"]

FROM base AS build

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sforum-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sforum-worker ./cmd/worker

FROM alpine:3.22 AS api

RUN apk add --no-cache ca-certificates
COPY --from=build /out/sforum-api /usr/local/bin/sforum-api
EXPOSE 8080
CMD ["sforum-api"]

FROM alpine:3.22 AS worker

RUN apk add --no-cache ca-certificates
COPY --from=build /out/sforum-worker /usr/local/bin/sforum-worker
CMD ["sforum-worker"]
