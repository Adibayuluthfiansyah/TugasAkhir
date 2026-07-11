FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dinsos-backend main.go
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1001 appuser
WORKDIR /app
COPY --from=builder /app/dinsos-backend .
USER appuser
EXPOSE 8000
CMD ["./dinsos-backend"]