# Stage 1: Build the Go application
FROM golang:1.22 AS builder

# Set the working directory
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o api .

# Stage 2: Create the runtime image
FROM alpine:latest

# Install CA certificates (needed for HTTPS requests)
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/api .

# Expose the port the API runs on
EXPOSE 8080

# Run the application
CMD ["./api"]
