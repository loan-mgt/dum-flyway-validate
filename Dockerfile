# Use a lightweight Alpine image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Install required tools
RUN apk --update add curl git

# Build argument for dum-flyway-validate version (mandatory)
ARG DUM_FLYWAY_VALIDATE_VERSION
# Validate that the build argument is provided
RUN test -n "$DUM_FLYWAY_VALIDATE_VERSION" || (echo "Build argument DUM_FLYWAY_VALIDATE_VERSION is required" && exit 1)

# Download dum-flyway-validate
RUN curl -LO https://github.com/Qypol342/dum-flyway-validate/releases/download/$DUM_FLYWAY_VALIDATE_VERSION/dum-flyway-validate
RUN chmod +x dum-flyway-validate

