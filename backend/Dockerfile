# Stage 1: Build binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install certificates and build dependencies
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/spark-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/spark-bot ./cmd/bot

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app
RUN apk upgrade --no-cache && apk --no-cache add ca-certificates tzdata

RUN addgroup -S -g 10001 spark && adduser -S -u 10001 -G spark -H -s /sbin/nologin spark

COPY --from=builder --chmod=0555 /app/spark-server /app/spark-server
COPY --from=builder --chmod=0555 /app/spark-bot /app/spark-bot
COPY --chmod=0755 media /app/media
RUN chmod -R a+rX /app/media

USER spark:spark

EXPOSE 8080

CMD ["/app/spark-server"]
