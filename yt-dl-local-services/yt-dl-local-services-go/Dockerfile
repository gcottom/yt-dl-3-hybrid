FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN go build -o downloader ./cmd/downloader/
RUN go build -o server ./cmd/server/
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/downloader .
COPY --from=builder /app/server .
ENTRYPOINT ["./server"]
