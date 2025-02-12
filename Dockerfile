# Build Stage
FROM golang:1.24.0-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT=""

ARG VERSION="development"
ARG BUILDTIME=""
ARG REVISION=""

# Set necessary environment variables for Go cross-compilation
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT}

# Update CA Certs
RUN apk --update --no-cache add ca-certificates

# Set the working directory inside the container
WORKDIR /app

# Copy the source code
COPY . .

# Download Go modules
RUN go mod tidy

# Build the Go application
RUN go build -o dras main.go

# Final Stage
FROM scratch AS final

LABEL \
    org.opencontainers.image.title="dras" \
    org.opencontainers.image.source="https://github.com/jacaudi/dras"

# Copy the built binary from the build stage
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /app/dras /dras

# Command to run the application
CMD ["/dras"]