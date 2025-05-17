FROM golang:1.23.4-alpine AS builder

# Install git and build-essential packages
RUN apk add --no-cache git gcc musl-dev

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/api/main.go

FROM alpine:3.19

# Install chromium for go-rod
RUN apk add --no-cache chromium ca-certificates tzdata

# Set environment variables
ENV CHROME_PATH=/usr/bin/chromium-browser
ENV TZ=UTC

# Create a non-root user to run the application
RUN adduser -D -g '' appuser
USER appuser

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/main .
# Copy any additional required files
COPY --from=builder /app/.env .

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./main"]
