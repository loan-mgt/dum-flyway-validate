# Use a lightweight Alpine image with Go installed
FROM golang:alpine AS builder

# Set the working directory
WORKDIR /app

# Copy the source code into the image
COPY main.go .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o dum-flyway-validate -ldflags="-w -s" main.go

# Use a minimal Alpine image for the final image
FROM alpine:latest

# Install required tools
RUN apk --update add curl git \
    && rm -rf /var/cache/apk/*

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/dum-flyway-validate .

# Add /app to the PATH
ENV PATH="/app:${PATH}"

# Add a non-root user for better security
RUN adduser -D -g '' appuser
USER appuser

# Specify the command to run on container start
ENTRYPOINT ["./dum-flyway-validate"]
CMD ["--help"]