# ads-analyzer

A small, production‑ready Go (1.24) service that analyzes `ads.txt` files for one or many domains. It returns advertiser domains and their counts, supports batch analysis, pluggable caching (memory/Redis), per‑client rate limiting, Prometheus metrics, structured/rotating logs, health/readiness probes, and build metadata via `/version`.

---

## Quick start

### Requirements
- **Go 1.24**
- Docker 24+ (optional, for containerized runs)
- Redis 7+ (optional, when using the Redis cache backend)
- GNU Make (optional; you can run the raw commands instead)

### Clone & run (memory cache)
```bash
# from the project root
LOG_LEVEL=info PORT=8080 go run ./cmd/server
```
Now:
```bash
curl -s http://localhost:8080/health | jq        # {"status":"ok"}
curl -s http://localhost:8080/ready  | jq        # {"status":"ready"}
curl -s "http://localhost:8080/api/analysis?domain=msn.com" | jq
```

### Run with Redis via docker‑compose
```bash
docker compose up --build -d
curl -s http://localhost:8080/ready | jq   # proves cache path works
```

> **Windows note**: if `-race` is troublesome locally, use WSL2 or Docker for race tests (see Testing below).

---

## Configuration
All configuration is 12‑Factor via environment variables. `.env` is **optional** and only for local dev (missing `.env` will not crash the service).

### Environment variables
```dotenv
# --- Server ---
PORT=8080
LOG_LEVEL=info                     # debug|info|warn|error

# --- Logging ---
LOG_OUTPUT=stdout                  # stdout|file|both
LOG_FILE_PATH=./logs/ads-analyzer.log
LOG_FILE_MAX_SIZE=50               # MB
LOG_FILE_MAX_BACKUPS=5
LOG_FILE_MAX_AGE=28                # days
LOG_FILE_COMPRESS=true

# --- Metrics ---
METRICS_ENABLED=true               # expose /metrics

# --- Fetcher ---
FETCH_TIMEOUT=5s                   # per request timeout
HTTP_FALLBACK=true                 # try http:// if https:// fails

# --- Cache ---
CACHE_BACKEND=memory               # memory|redis
CACHE_TTL=10m
# Redis (only when CACHE_BACKEND=redis)
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0

# --- Rate limit (token bucket per client) ---
RATE_PER_SEC=10
RATE_BURST=20

# --- Batch ---
BATCH_WORKERS=8
```
See `.env.example` for a curated example.

---

## Endpoints
- `GET /health` → `{ "status": "ok" }` (liveness)
- `GET /ready`  → `{ "status": "ready" }` after cache round‑trip (readiness)
- `GET /metrics` → Prometheus metrics (enabled when `METRICS_ENABLED=true`)
- `GET /version` → build metadata `{ version, commit, build_time, go }`
- `GET /api/analysis?domain=<domain>` → single domain result
- `POST /api/batch-analysis` `{ "domains": ["msn.com","cnn.com"] }` → results array

Example batch call (bash):
```bash
curl -s -X POST http://localhost:8080/api/batch-analysis \
  -H 'Content-Type: application/json' \
  -d '{"domains":["msn.com","cnn.com","vidazoo.com"]}' | jq
```

---

## Observability
- **Logs**: zerolog JSON to stdout and/or file (with rotation via lumberjack).
  - File logging example:
    ```bash
    LOG_OUTPUT=both LOG_FILE_PATH=./logs/ads-analyzer.log go run ./cmd/server
    ```
- **Metrics**: Prometheus client (`/metrics`) with request/latency, cache hits/misses, fetch durations, and rate‑limit blocks.
- **Build info**: `/version` shows the git tag/commit/build time baked at build time.

### Rate‑limit demo
```bash
RATE_PER_SEC=1 RATE_BURST=1 go run ./cmd/server &
# First request → 200
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health
# Immediate second → 429
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health
# Metric increments
curl -s http://localhost:8080/metrics | grep rate_limit_blocks_total
```

---

## Build metadata (/version) in Docker & Compose
The Dockerfile computes the Go module path inside the build container and injects metadata via ldflags.

**Dockerfile args**: `VERSION`, `COMMIT`, `BUILD_TIME`.

### Docker (manual)
```bash
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

docker build \
  --build-arg VERSION="$VERSION" \
  --build-arg COMMIT="$COMMIT" \
  --build-arg BUILD_TIME="$BUILD_TIME" \
  -t ads-analyzer:latest .

docker run --rm -p 8080:8080 ads-analyzer:latest
curl -s http://localhost:8080/version | jq
```

### docker‑compose
`docker-compose.yml` forwards the args via environment variables:
```yaml
services:
  api:
    build:
      context: .
      args:
        VERSION: ${VERSION:-dev}
        COMMIT: ${COMMIT:-none}
        BUILD_TIME: ${BUILD_TIME:-unknown}
```
Export them before building:
```bash
export VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
export COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo none)
export BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

docker compose build --no-cache api && docker compose up -d
curl -s http://localhost:8080/version | jq
```

---

## Testing

### Unit tests
```bash
go test ./...
```

### Race detector
- **Linux/macOS/WSL2**:
  ```bash
  go test ./... -race -count=1
  ```
- **Windows native**: race builds require a working MinGW‑w64 toolchain and can be flaky. Prefer WSL2 or Docker for `-race`:
  ```bash
  docker run --rm -v ${PWD}:/src -w /src golang:1.22 \
    bash -lc "apt-get update && apt-get install -y gcc && go test ./... -race -count=1"
  ```

### Manual checks
```bash
# cache behavior (memory)
CACHE_BACKEND=memory CACHE_TTL=5s go run ./cmd/server &
curl -s "http://localhost:8080/api/analysis?domain=msn.com" | jq   # cached=false
curl -s "http://localhost:8080/api/analysis?domain=msn.com" | jq   # cached=true
sleep 6
curl -s "http://localhost:8080/api/analysis?domain=msn.com" | jq   # cached=false again
```

---

## Makefile targets (optional)
- `tidy` — `go mod tidy`
- `vet` — `go vet ./...`
- `test` — `go test ./...`
- `race` — `go test ./... -race -count=1`
- `build` — build binary with ldflags → `./bin/ads-analyzer`
- `run` — run with ldflags
- `docker` — docker build with VERSION/COMMIT/BUILD_TIME
- `compose-build` — build compose service with metadata
- `compose-up` / `compose-down`

> If you don’t use Make, copy the equivalent commands shown above.

---

## CI (GitHub Actions)
- Uses `actions/setup-go@v5` with **Go 1.24.x** to match the project toolchain.
- Steps: `tidy` → `vet` → `test -race` (Linux) → `build with ldflags`.
- If you previously saw noisy tar cache warnings, ensure CI runs on 1.24.x (or set `GOTOOLCHAIN=local` if you intentionally stay on an older Go).

---

## Design highlights
- **Transport vs domain vs infra**: HTTP concerns (handlers/middlewares) isolated from analysis logic and infra (cache, rate limit, logging, metrics).
- **Small interfaces** (`Cache`, `Fetcher`, `Analyzer`) for testability and substitution.
- **Token‑bucket rate limiter** (per client via API key or IP) to protect the service.
- **Caching**: memory or Redis with the same JSON payloads.
- **Observability**: structured logs, metrics, probes, and build info.

---

## Troubleshooting
- `/version` shows `dev/none/unknown` → you built without passing build args/ldflags. Use the Docker/Compose instructions above.
- `429 Too Many Requests` immediately → your `RATE_PER_SEC`/`RATE_BURST` are low; raise them.
- `GET /ready` returns non‑ready → check `CACHE_BACKEND` and `REDIS_ADDR`; verify Redis is reachable.
- Windows `-race` errors → use WSL2 or Docker for race tests.

---

## License
MIT (unless otherwise noted).
