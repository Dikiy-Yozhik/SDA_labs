# Stage 1: compile the Go application
FROM golang:1.26-alpine3.24 AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/weather-api ./cmd/weather-api

# Stage 2: run the compiled application
FROM alpine:3.24.2

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=builder /out/weather-api /usr/local/bin/weather-api

USER app

EXPOSE 8080

ENTRYPOINT ["weather-api"]