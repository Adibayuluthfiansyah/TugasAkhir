FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o dinsos-backend main.go
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/dinsos-backend .
EXPOSE 8000

CMD ["./dinsos-backend"]