FROM golang:1.25.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/proxy-server main.go

FROM alpine:3.20

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

COPY --from=builder /app/proxy-server /app/proxy-server

EXPOSE 9364

ENV CLICKHOUSE_HOST=clickhouse-server
ENV CLICKHOUSE_PORT=9000

ENTRYPOINT ["/app/proxy-server"]