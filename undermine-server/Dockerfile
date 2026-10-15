FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

FROM alpine:3.24

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata sqlite-libs

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

RUN adduser -D -g '' appuser
RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

COPY --from=builder /app/server /app/server

EXPOSE 8080

ENTRYPOINT [ "/app/server"]