FROM golang:1.24-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/api

FROM golang:1.24-alpine
WORKDIR /app
RUN apk add --no-cache ca-certificates wget git
COPY --from=builder /app/app /app/app
COPY --from=builder /go/bin/ /go/bin/
COPY --from=builder /usr/local/go /usr/local/go
COPY --from=builder /app/go.mod /app/go.mod
COPY --from=builder /app/go.sum /app/go.sum
COPY --from=builder /app/internal /app/internal
COPY --from=builder /app/cmd /app/cmd
COPY migrate.sh /app/migrate.sh
RUN chmod +x /app/migrate.sh
ENV PATH="/go/bin:/usr/local/go/bin:$PATH"
ENV GIN_MODE=release
EXPOSE 8080
CMD ["/app/app"]
