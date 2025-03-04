# Use the official Golang image to build the binary
FROM golang:1.23 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o blackandwhite main.go

# Start a new stage from scratch
FROM gcr.io/distroless/base-debian12

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/blackandwhite /app/blackandwhite

# Set the working directory
WORKDIR /app

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["./blackandwhite"]
