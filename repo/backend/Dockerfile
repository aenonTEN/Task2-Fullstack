FROM golang:1.24-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/api

FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /app/app /app/app
COPY migrate.sh /app/migrate.sh
RUN chmod +x /app/migrate.sh
EXPOSE 8080
CMD ["/app/app"]
