# Stage 1: Build binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install certificates and build dependencies
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod ./
# If go.sum exists, copy it, otherwise download modules
RUN go mod download || true

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/spark-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/spark-bot ./cmd/bot

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/spark-server /app/spark-server
COPY --from=builder /app/spark-bot /app/spark-bot

EXPOSE 8080

CMD ["/app/spark-server"]
