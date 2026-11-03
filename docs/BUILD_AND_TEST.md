# Build, Test & Deployment Guide

This guide covers building, testing, configuring, and operating MintSpace across local and production environments.

---

## 1. System Requirements

* **Go**: Version 1.23 or newer.
* **CGO**: **Not required**. MintSpace uses a pure-Go SQLite driver (`modernc.org/sqlite`), enabling seamless native compilation on any OS without GCC.
* **Docker & Docker Compose** (optional): For containerized multi-node cluster deployment.

---

## 2. Local Development

### Clone and Run
```bash
git clone https://github.com/Supriyo-455/mintspace.git
cd mintspace

# Download dependencies
go mod download

# Run directly
go run .
```

The server binds to `http://127.0.0.1:3000`.

### Default Credentials
Upon first launch, an editorial administrator account is automatically provisioned:
* **Email**: `admin@mintspace.io`
* **Password**: `admin123`

---

## 3. Automated Test Suite

MintSpace includes a test suite covering isolated database migrations, Content-Addressable Storage (CAS), access control, rate limiting, and peer mesh resolution:

```bash
# Run all unit and integration tests with verbose output
go test -v ./...

# Run tests with race condition detector
go test -race -v ./...

# Run test coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Verified Test Suites:
* `TestCommentDeletion`: Verifies comment retraction, ownership authorization, and cascade removal of replies.
* `TestArchiveAndPaperView`: Validates archive listing, manuscript rendering, and metadata presentation.
* `TestWriteBlogAndCASVerification`: Tests Markdown submission, SHA-256 calculation, and CAS blob creation.
* `TestLikesAndComments`: Validates endorsement toggles and threaded recursive commentary trees.
* `TestPrivateBlogRestrictions`: Tests authorization boundaries between public and restricted papers.
* `TestAdminConsole`: Verifies editorial console RBAC and user role toggling.
* `TestHealthzEndpoint`: Validates health and cluster status metrics.
* `TestSecurityHeaders`: Checks strict CSP, X-Frame-Options, and nosniff headers.
* `TestPathTraversalProtection`: Validates strict rejection of malformed or traversal hashes (`../../`).
* `TestRateLimiter`: Tests token-bucket burst and throttling mechanics.
* `TestDistributedCASMeshSync`: Validates cross-node peer resolution, hash verification, and self-healing storage.

---

## 4. Compiling Production Binaries

### Native Build
```bash
# Compile stripped, optimized static binary
go build -ldflags="-s -w" -o mintspace .
```

### Cross-Compilation (No GCC required)
```bash
# Linux x86_64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mintspace-linux-amd64 .

# Linux ARM64 (Raspberry Pi / AWS Graviton)
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o mintspace-linux-arm64 .

# Windows x86_64
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o mintspace-windows.exe .

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o mintspace-darwin-arm64 .
```

---

## 5. Docker & Multi-Node Cluster Deployment

### Building the Docker Image
```bash
docker build -t mintspace:latest .
```

### Running the Multi-Node Cluster
The included `docker-compose.yml` launches a 2-node self-healing cluster behind a Caddy reverse proxy with automatic SSL:

```bash
docker compose up -d
```

### Cluster Endpoints:
* `http://localhost`: Caddy load balancer routing across nodes.
* `http://localhost:3000/healthz`: Health check, active storage mode, and cluster peers.
* `http://localhost:3000/api/v1/sync/blob`: Distributed CAS peer synchronization endpoint.

---

## 6. Configuration & Environment Variables

| Variable | Default | Purpose |
| :--- | :--- | :--- |
| `SECRET` | Auto-generated | Cryptographic secret key used to sign and verify JWT authentication tokens. |
| `PEER_NODES` | None | Comma-separated list of peer URLs (e.g. `http://node2:3000,http://node3:3000`). |
| `PORT` | `3000` | Port on which the HTTP server listens. |

File-based configuration is stored in [`config.json`](file:///c:/Users/pauls/Downloads/mintspace/config.json):
```json
{
  "Address": "127.0.0.1:3000",
  "Mode": "Localhost"
}
```
