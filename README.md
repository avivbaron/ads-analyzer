## ads-analyzer — ads.txt REST API (Go)

A production-minded Go service that fetches and parses `https://<domain>/ads.txt`, counts seller (advertiser) domains, and returns a structured JSON report. Supports batch analysis, pluggable cache backends (memory, Redis), and a custom in-house per-client rate limiter. Ships with Docker, Compose, and CI.

### Features
- **Single analysis**: `GET /api/analysis?domain=msn.com`
- **Batch analysis**: `POST /api/batch-analysis` with `{ "domains": ["msn.com", "cnn.com"] }`
- **Robust parsing**: ignores comments/directives, handles inline `#`, trims whitespace
- **Pluggable cache**: memory (default) or Redis (via env), JSON-encoded entries, TTL
- **Rate limiting**: custom token-bucket per IP or `X-API-Key` (no external libs)
- **Time-bounded fetch**: HTTPS first, optional HTTP fallback, redirect-safe
- **Prod bits**: structured logs, graceful shutdown, Dockerfile, docker-compose, GitHub Actions CI

### Quickstart
# ads-analyzer — ads.txt REST API (Go)

A production-minded Go service that fetches and parses `https://<domain>/ads.txt`, counts seller (advertiser) domains, and returns a structured JSON report. Supports **batch analysis**, **pluggable cache** (memory/Redis), custom **rate limiting**, and **Prometheus metrics**.

---

## ✨ Features
- **Single analysis**: `GET /api/analysis?domain=msn.com`
- **Batch analysis**: `POST /api/batch-analysis` with `{ "domains": ["msn.com", "cnn.com"] }`
- **Robust parser**: ignores comments/directives, strips inline `#`, normalizes domains
- **Caching**: memory (default) or Redis; JSON-encoded entries with TTL
- **Custom rate limiting**: per IP or `X-API-Key` (token bucket; no external limiter libs)
- **Observability**: structured logs (stdout/file), Prometheus `/metrics`, health `/healthz`, readiness `/readyz`
- **Docker & CI**: Dockerfile, docker-compose, GitHub Actions

---

## 📦 Requirements
- **Go**: 1.22+
- **Network**: outbound HTTPS/HTTP to fetch `ads.txt`
- **Optional**: Docker & Docker Compose (if running via containers)
- **Optional**: Redis 7+ (if `CACHE_BACKEND=redis`)

---

## ⚙️ Configuration (env vars)
| Var | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `LOG_LEVEL` | `info` | `debug|info|warn|error` |
| `LOG_OUTPUT` | `stdout` | `stdout|file|both` |
| `LOG_FILE_PATH` | `./logs/ads-analyzer.log` | log file when `file`/`both` |
| `LOG_FILE_MAX_SIZE` | `50` | rotate size (MB) |
| `LOG_FILE_MAX_BACKUPS` | `5` | rotated file count |
| `LOG_FILE_MAX_AGE` | `28` | days to keep |
| `LOG_FILE_COMPRESS` | `true` | gzip rotated logs |
| `FETCH_TIMEOUT` | `5s` | ads.txt fetch timeout |
| `HTTP_FALLBACK` | `true` | try `http://` if `https://` fails |
| `CACHE_BACKEND` | `memory` | `memory|redis` |
| `CACHE_TTL` | `10m` | default TTL for cached results |
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis address |
| `REDIS_PASSWORD` | `` | Redis password |
| `REDIS_DB` | `0` | Redis DB index |
| `RATE_PER_SEC` | `10` | allowed requests/sec per client |
| `RATE_BURST` | `20` | token bucket burst size |
| `BATCH_WORKERS` | `8` | worker pool size for batch endpoint |
| `METRICS_ENABLED` | `true` | expose `/metrics` |

Copy `.env.example` to `.env` and edit as needed.

---

## ▶️ Run locally

### macOS / Linux (bash/zsh)
```bash
# from repo root
export LOG_LEVEL=debug PORT=8080
export CACHE_BACKEND=memory CACHE_TTL=10m
go run ./cmd/server
```

### Windows — PowerShell
```powershell
$env:LOG_LEVEL="debug"; $env:PORT="8080";
$env:CACHE_BACKEND="memory"; $env:CACHE_TTL="10m";
go run ./cmd/server
```

### Windows — cmd.exe
```bat
set LOG_LEVEL=debug
set PORT=8080
set CACHE_BACKEND=memory
set CACHE_TTL=10m
go run .\cmd\server
```

---

## 🐳 Run with Docker
```bash
docker build -t ads-analyzer:latest .
docker run --rm -p 8080:8080 \
  -e LOG_LEVEL=info -e CACHE_BACKEND=memory \
  ads-analyzer:latest
```

### Docker Compose (with Redis ready)
```bash
docker compose up --build
```
- API at `http://localhost:8080`
- Redis at `localhost:6379` (if enabled in compose)

---

## 🔌 API

### Single domain
```http
GET /api/analysis?domain=msn.com
```
**Response 200**
```json
{
  "domain": "msn.com",
  "total_advertisers": 189,
  "advertisers": [
    {"domain":"google.com","count":102},
    {"domain":"appnexus.com","count":60},
    {"domain":"rubiconproject.com","count":27}
  ],
  "cached": false,
  "timestamp": "2025-07-13T10:30:45Z"
}
```

### Batch
```http
POST /api/batch-analysis
Content-Type: application/json

{"domains":["msn.com","cnn.com","vidazoo.com"]}
```
**Response 200**
```json
{"results": [ /* one AnalysisResult per domain, in order */ ]}
```

### Status codes
- `200` OK — success
- `400` Bad Request — missing/invalid domain
- `404` Not Found — `ads.txt` not found
- `429` Too Many Requests — rate limit exceeded
- `502` Bad Gateway — upstream/network error
- `504` Gateway Timeout — fetch timeout

---

## 📈 Observability
- **Metrics (Prometheus)**: `GET /metrics`
  - `http_requests_total{method,path,status}`
  - `http_request_duration_seconds{method,path,status}`
  - `cache_hits_total{op}`, `cache_misses_total{op}`
  - `fetch_duration_seconds{scheme}`
  - `rate_limit_blocks_total{path}`
- **Health**: `GET /health` (liveness)
- **Readiness**: `GET /ready` (cache Set/Get/Delete round-trip)

Prometheus scrape example:
```yaml
scrape_configs:
  - job_name: 'ads-analyzer'
    static_configs:
      - targets: ['host.docker.internal:8080']
```

---

## 🧪 Testing
```bash
go test ./... -race
# only cache tests
go test ./internal/cache
# Redis integration (skips if Redis unavailable)
REDIS_ADDR=127.0.0.1:6379 go test ./internal/cache -run Redis -v
```

---

## 🧰 Makefile (handy targets)
```makefile
run:      ## Run locally
	LOG_LEVEL=debug PORT=8080 go run ./cmd/server

test:     ## Run unit tests
	go test ./...

vet:      ## Go vet
	go vet ./...

tidy:     ## Go mod tidy
	go mod tidy

build:    ## Build binary
	go build ./cmd/server

docker:   ## Build container image
	docker build -t ads-analyzer:latest .

docker-run: ## Run container
	docker run --rm -p 8080:8080 -e LOG_LEVEL=info ads-analyzer:latest

compose-up: ## Up with Docker Compose
	docker compose up --build

compose-down: ## Down Compose
	docker compose down
```

---

## 🛟 Troubleshooting
- `429 Too Many Requests`: lower your call rate or raise `RATE_PER_SEC`/`RATE_BURST`.
- `504 Gateway Timeout`: increase `FETCH_TIMEOUT` or check target site availability.
- Empty results: ensure the target serves a valid `ads.txt` and your network allows outbound HTTP(S).
- Redis backend: confirm `REDIS_ADDR` reachable; try `redis-cli -h 127.0.0.1 -p 6379 PING`.

---
