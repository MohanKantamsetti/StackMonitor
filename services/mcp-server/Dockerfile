FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/mcp-server .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/mcp-server .

EXPOSE 5001

CMD ["./mcp-server"]
