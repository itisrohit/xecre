# STAGE 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy and download dependencies first (for faster caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build exactly what we need
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o xecre-api ./cmd/api/main.go

# STAGE 2: Deployment
FROM alpine:3.19

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/xecre-api .

# Expose port and start
EXPOSE 8080
CMD ["./xecre-api"]
