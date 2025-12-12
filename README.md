Balto: Developer-Focused API Gateway & Load Balancer
=====================================================

Balto is a lightweight, developer-friendly HTTP reverse proxy and load balancer written in Go. It aims to give you fast, predictable routing based on host and path, with simple configuration, clean code, and a clear operational story. For observability, Balto exposes Prometheus metrics that you can visualize with Grafana.

At this stage, Balto is under active development. The core router and proxy are in place with tests; more features like health checks, TLS termination and the full dashboard will follow.


What Balto does today
---------------------
- Host and path routing with:
  - exact segments (e.g. `/api`)
  - parameters (e.g. `/users/:id`)
  - wildcard segments (e.g. `/static/*`)
- Hot-reload routing using an atomic, immutable router (no downtime).
- Reverse proxy with:
  - Prefix stripping (supports wildcard prefixes like `/api/v1/*`)
  - Streaming request/response bodies
  - Context cancellation (client disconnect cancels upstream)
  - Forwarded headers (`X-Forwarded-For`, `X-Forwarded-Proto`, `X-Forwarded-Host`)
  - Route parameters propagated as `X-Param-<name>` headers (e.g. `X-Param-id`)
- Minimal HTTP server with `/health` endpoint.
- CI pipeline for tests and lint; local pre-commit hooks for format and lint.


What we aim to achieve next
---------------------------
- Multiple load balancing algorithms (round-robin, least-connections, weighted, etc.).
- Active health checks and automatic reintegration of healthy backends.
- TLS termination (Let’s Encrypt), per-backend TLS options.
- Metrics (request rates, latency, error rates) exposed via Prometheus (Grafana dashboards).
- Simple YAML configuration with live reload and validation.
- Production-ready Docker images and Kubernetes support.


Project layout
--------------
- `cmd/balto` — main entrypoint (HTTP server, `/health`)
- `internal/router` — immutable routing tree (host + path, params, wildcard)
- `internal/proxy` — HTTP reverse proxy
- `internal/server` — HTTP server wrapper with timeouts and graceful shutdown
- `configs/` — configuration skeleton (`balto.config.yaml`, `services/`)
- `.github/workflows/ci.yml` — CI for tests, lint, and formatting check
- `Makefile` — build, test, run, lint, lint-fix, format, install-hooks
- `scripts/` — local setup and Git hook installer


Prerequisites
-------------
- Go 1.23+
- Make
- git
- For linting locally: `golangci-lint` (CI runs it regardless)
- For the full observability stack: Docker + Docker Compose

Install `golangci-lint` (optional but recommended for local hooks):

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc
source ~/.zshrc
```


Quick start (local)
-------------------
Clone and build:

```bash
git clone https://github.com/diabeney/balto.git && cd balto
make build
```

Run the server:

```bash
make run
# Server listens on :5500 (see configs/balto.config.yaml)
curl -i http://localhost:5500/health
```

Observability (Docker Compose: Balto + Prometheus + Grafana)
------------------------------------------------------------

```bash
docker compose up --build
```

- Watch Balto logs:
  - `docker compose logs -f --tail=200 balto`
  - or `make compose-logs`

- Upstream addressing in Docker:
  - If your `configs/balto.config.yaml` uses port-only targets like `"8080"`, Balto will probe/route to `${BALTO_UPSTREAM_HOST}:8080`.
  - In `docker-compose.yml` we set `BALTO_UPSTREAM_HOST=host.docker.internal` so Balto can reach services running on your host machine.
  - If your upstreams are other containers in the same compose network, set `ports` to `service-name:port` (e.g. `"users-api:8080"`) or full URLs (e.g. `"http://users-api:8080"`).

- Balto: `http://localhost:5500`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (login: `admin` / `admin`)

Prometheus scrapes Balto at `GET /balto/metrics`.

What you’ll see in Grafana
--------------------------

- **Balto (application)**:
  - Request rate, status code rate, latency (p95), TTFB (p95)
  - Per-service request rate as a **stacked area** graph (service == `route` label)
  - Per-backend request rate as a **stacked area** graph (filtered by service)
  - Balto process CPU and memory (via Prometheus `process_*` / `go_*` collectors)

- **System / container**:
  - CPU, memory, disk from **node-exporter**
  - Container metrics from **cAdvisor** (optional; useful when you run upstreams in Docker too)

Note on “host machine” metrics
------------------------------

If you’re on Docker Desktop (macOS/Windows), node-exporter typically reflects the **Docker VM**, not the full host OS. For production-grade host observability, run node-exporter on the host (or use Grafana Agent / OpenTelemetry Collector) and scrape that instead.

Grafana “graphs-only” view (kiosk mode)
---------------------------------------

- Default home dashboard is provisioned to **Balto**.
- For a graphs-only view (no sidebar), open:
  - `http://localhost:3000/d/balto/balto?kiosk`

Run tests:

```bash
make test
```

Format and lint (local):

```bash
make format
make lint
# or auto-fix what can be fixed
make lint-fix
```


Git hooks (format + lint on commit)
-----------------------------------
Install the pre-commit hook that runs `go fmt` and `golangci-lint`:

```bash
make install-hooks
```

If lint is not found in your PATH, the hook will guide you to install it and add your Go bin to PATH. CI will still enforce linting even if you skip local hooks.


How routing works (short version)
---------------------------------
- You define routes per host with path prefixes like:
  - `/` (root)
  - `/api`
  - `/users/:id`
  - `/static/*`
- Matching is strict unless a wildcard is explicit. For example:
  - `/api` does not automatically match `/api/v1` unless you use `/api/*` or define `/api/v1`.
- The proxy strips the matched prefix before forwarding. With `/api/v1/*` and a request to `/api/v1/users/123`, the backend sees `/users/123`.
- Path params are attached as headers: a route `/users/:id` adds `X-Param-id` with the matched value.


Configuration (current state)
-----------------------------
- Global config skeleton: `configs/balto.config.yaml` (logging, timeouts, TLS toggles, etc.).
- Service configs folder: `configs/services/` (reserved; to be wired).
- Code already contains `BuildFromConfig` for basic host/path + ports. A full config loader/CLI wiring is planned.

Example (what configuration will look like):

```yaml
global:
  load_balancing:
    algorithm: round-robin
  timeouts:
    read: 5s
    write: 5s
    idle: 30s
services:
  - domain: example.com
    path_prefix: /api/v1/*
    ports: ["8081", "8082"]
```


Commands you’ll use often
-------------------------
- `make build` — compile the binary to `bin/balto`
- `make run` — run the server from sources
- `make test` — run all tests
- `make lint` — run linter
- `make lint-fix` — auto-fix what can be fixed
- `make format` — go fmt all packages
- `make install-hooks` — install local pre-commit hook


Continuous Integration
----------------------
GitHub Actions runs on each push/PR:
- `go mod tidy`
- formatting check (`gofmt -s -l`)
- `go test ./...`
- `golangci-lint` (latest action)

The pipeline fails if formatting or linting is not clean.


Development notes
-----------------
- Router is immutable; `Add` returns a new router. This keeps hot reloads safe.
- Proxy streams responses and respects client cancellations using `context.Context`.
- Timeouts are configured in the HTTP server to keep connections responsive.
- Tests exist for routing, server timeouts, and proxy behaviour (prefix stripping, params).


Roadmap
-------
- Health checking and backend pool management
- TLS termination and certificate management
- richer metrics and a real-time dashboard
- load balancing strategies and sticky sessions
- production-grade config loader and CLI


License
-------
Apache-2.0. See `LICENSE`.


