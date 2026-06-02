# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o cozy-backend cmd/api/main.go

# Production Run Stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/cozy-backend .

# Expose Go API port
EXPOSE 8080

CMD ["./cozy-backend"]
