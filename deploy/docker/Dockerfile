FROM golang:1.26.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o aegis ./cmd/aegis
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o aegis-cli ./cmd/aegis-cli

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/aegis .
COPY --from=builder /app/aegis-cli .

EXPOSE 7000

CMD ["./aegis"]