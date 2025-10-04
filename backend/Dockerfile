# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy all source code including vendor
COPY . .

# Build the application using vendored dependencies
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o main .

# Final stage - use a minimal golang alpine image
FROM golang:1.24-alpine

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/.env .

EXPOSE 8080

CMD ["./main"]
