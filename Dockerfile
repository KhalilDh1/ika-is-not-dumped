# Use a Go base image that supports Go 1.23
FROM golang:1.23-alpine

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Install dependencies
RUN go mod tidy

# Copy the rest of the application code
COPY . .

# Build the Go app
RUN go build -o main .

# Expose the port the app will run on
EXPOSE 8080

# Run the Go app
CMD ["./main"]
