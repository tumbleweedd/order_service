# Use a base image that includes Go runtime and tools
FROM golang:1.22.2-alpine3.19 as builder

# Install Git
RUN apk update && apk add --no-cache git

# Set the working directory
WORKDIR /go/src/app

# Copy go.mod and go.sum files to cache dependencies
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/app ./cmd/orderService/

# Final stage
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage to the final image
COPY --from=builder /go/bin/app /app/app

# Copy the config file from the host machine into the container
COPY config/config.yaml /app/config.yaml

# Set the environment variable for the configuration path
ENV CONFIG_PATH="/app/config.yaml"

# Command to run the application
CMD ["/app/app"]
