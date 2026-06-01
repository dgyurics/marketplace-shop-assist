FROM golang:1.24-alpine3.22 AS builder
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -o shop-assist .

FROM alpine:3.22

WORKDIR /app

# Copy binary
COPY --from=builder /app/shop-assist .

EXPOSE 8080
CMD ["./shop-assist"]