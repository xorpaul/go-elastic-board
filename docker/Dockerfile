# syntax=docker/dockerfile:1

########################
# Build stage
########################
FROM golang:1.24.5-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata git

# Go modules first (cache-friendly)
COPY go.mod go.sum ./
RUN go mod download

# App source (includes embedded static assets)
COPY static ./static
COPY *.go ./

# Optional version metadata
ARG BUILD_VERSION=dev
ARG BUILD_TIME=local

ENV CGO_ENABLED=0
RUN go build -trimpath \
  -ldflags "-s -w -X main.buildversion=${BUILD_VERSION} -X main.buildtime=${BUILD_TIME}" \
  -o /out/go-elastic-board

########################
# Runtime stage
########################
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=build /out/go-elastic-board /app/go-elastic-board

EXPOSE 8080
USER appuser
ENTRYPOINT ["/app/go-elastic-board"]
CMD ["-config", "/etc/go-elastic-board/config.yaml"]

