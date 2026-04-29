# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/ventopanel ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 65532 nonroot && \
    adduser -D -u 65532 -G nonroot nonroot && \
    mkdir -p /data/backups && \
    chown nonroot:nonroot /data/backups

WORKDIR /app

COPY --from=builder /out/ventopanel /app/ventopanel
COPY --from=builder /app/migrations /app/migrations

USER nonroot

EXPOSE 8080

ENTRYPOINT ["/app/ventopanel"]
