FROM golang:1.25-alpine AS builder

# Set working directory 
WORKDIR /app

# Install build tools
RUN apk add --no-cache git

# Copy go.mod and go.sum first for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary 
RUN go build -o risk-based-authn ./cmd/api

# Create minimal runtime image
FROM alpine:latest 

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/risk-based-authn .
COPY rules.yaml .

# Expose the port the app listens on
EXPOSE 8080

# Run the binary 
CMD ["./risk-based-authn"]