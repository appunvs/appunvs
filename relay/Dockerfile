# syntax=docker/dockerfile:1.7
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/relay ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 relay
USER relay
WORKDIR /app
COPY --from=build /out/relay /app/relay
COPY config/config.yaml /app/config/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/relay", "-config", "/app/config/config.yaml"]
