# syntax=docker/dockerfile:1

# Builder Stage
FROM golang:alpine AS builder
WORKDIR /app

# Install dependencies for CGO if needed, though we aim for static
RUN apk add --no-cache git tzdata ca-certificates

# Download go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application statically
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/server ./cmd/server

# ---------------------------------------------
# Development Target
# ---------------------------------------------
FROM golang:alpine AS dev
WORKDIR /app
# Install tools like air for live-reloading if needed
RUN go install github.com/air-verse/air@latest
COPY . .
# Default dev command
CMD ["go", "run", "./cmd/server"]

# ---------------------------------------------
# Production Target (Hardened Distroless)
# ---------------------------------------------
FROM gcr.io/distroless/static-debian12:latest AS prod

# Copy binary from builder
COPY --from=builder /go/bin/server /server
# Copy timezones and CA certs
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV PORT=8080
ENV APP_ENV=prod
EXPOSE 8080

USER nonroot:nonroot

CMD ["/server"]
