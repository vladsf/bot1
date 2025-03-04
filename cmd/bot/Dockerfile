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
RUN go build -o bot1 main.go

# Start a new stage from scratch
FROM gcr.io/distroless/base-debian12

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/bot1 /app/bot1

# Set the working directory
WORKDIR /app

# Command to run the executable
ENTRYPOINT ["./bot1"]
