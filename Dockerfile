# ---- build stage ----
FROM golang:1.24-alpine AS build
ARG LDFLAGS=""
RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates
WORKDIR /src

# cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# copy sources
COPY . .

# static build for small final image
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags "-s -w ${LDFLAGS}" -o /out/ads-analyzer ./cmd/server

# ---- final stage ----
FROM scratch

# certs & timezone data for TLS and time ops
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

# binary
COPY --from=build /out/ads-analyzer /ads-analyzer

EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/ads-analyzer"]
