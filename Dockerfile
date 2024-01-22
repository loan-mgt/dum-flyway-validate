# Use a lightweight Alpine image with Go installed
FROM golang:alpine

# Set the working directory
WORKDIR /app

# Install required tools
RUN apk --update add curl git \
    && rm -rf /var/cache/apk/*

# Copy the source code into the image
COPY main.go .

# Build the binary
RUN go build -o dum-flyway-validate main.go && chmod +x dum-flyway-validate

# Add /app to the PATH
ENV PATH="/app:${PATH}"
