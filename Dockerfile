# ---- build stage ----
FROM golang:1.24-alpine AS build

# Build metadata injected by Compose/Make/CI (with sane defaults)
ARG VERSION="dev"
ARG COMMIT="none"
ARG BUILD_TIME="unknown"

RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates
WORKDIR /src

# cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# copy sources
COPY . .

# static build for a small final image
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
# Compute module name inside the container and apply ldflags
RUN MODULE=$(go list -m) && \
    go build -trimpath \
      -ldflags "-s -w \
        -X '${MODULE}/internal/buildinfo.Version=${VERSION}' \
        -X '${MODULE}/internal/buildinfo.Commit=${COMMIT}' \
        -X '${MODULE}/internal/buildinfo.BuildTime=${BUILD_TIME}'" \
      -o /out/ads-analyzer ./cmd/server

# ---- final stage ----
# Use Alpine instead of "scratch" so we can have curl for healthchecks (and certs/tzdata preinstalled)
FROM alpine:3.20

# Install runtime dependencies and curl for healthcheck
RUN apk add --no-cache ca-certificates tzdata curl && update-ca-certificates

# Copy the binary
COPY --from=build /out/ads-analyzer /usr/local/bin/ads-analyzer

# 65532 is a common "nonroot" uid/gid used in minimal images
USER 65532:65532

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["/usr/local/bin/ads-analyzer"]
