# syntax=docker/dockerfile:1

FROM node:24-alpine AS assets

WORKDIR /app

RUN npm install -g pnpm

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web ./web
RUN pnpm run css:build


FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

COPY --from=assets /app/web/static/css/app.css ./web/static/css/app.css

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/server \
    ./cmd/web

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/migrate \
    ./cmd/migrate


FROM alpine:3.21 AS runtime

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app

COPY --from=builder /out/server /app/server
COPY --from=builder /out/migrate /app/migrate

USER app

ENV APP_ENV=production
ENV APP_HOST=0.0.0.0
ENV PORT=8080

EXPOSE 8080

CMD ["/app/server"]