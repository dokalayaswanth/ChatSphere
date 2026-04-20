FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /chat-app ./main.go

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /chat-app /app/chat-app
COPY --from=builder /app/web /app/web

EXPOSE 8080
CMD ["/app/chat-app"]