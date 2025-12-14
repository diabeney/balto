# Balto

Balto is a high-performance, production-ready HTTP reverse proxy and API gateway designed for microservices architectures. It provides intelligent request routing, load balancing, circuit breaking, and health checking while maintaining operational simplicity and observability.

## Overview

Balto serves as a lightweight yet powerful edge proxy that sits between your clients and backend services. Unlike traditional API gateways that include business logic, Balto focuses on infrastructure concerns: routing, load balancing, resilience, and observability.

### Key Design Principles

- **Independent Operation**: Balto runs as a standalone service that can be deployed independently of your application stack
- **External Control Plane Integration**: Configuration and service management are handled by external control planes via REST APIs
- **Hot Reload**: Zero-downtime configuration updates through atomic router replacements
- **Observability First**: Comprehensive metrics and logging for operational visibility
- **Resilience**: Circuit breakers, health checking, and graceful degradation
- **Performance**: Optimized for low latency and high throughput

## Features

### Core Functionality

- **Intelligent Routing**: Host and path-based routing with parameter extraction
- **Load Balancing**: Multiple algorithms (round-robin, least-connections, weighted round-robin)
- **Circuit Breaking**: Automatic failure detection and recovery with configurable thresholds
- **Health Checking**: Active and passive health monitoring of backend services
- **Metrics & Monitoring**: Prometheus metrics with Grafana dashboards
- **Configuration Management**: Dynamic service registration and updates via REST API

### Routing Capabilities

- Exact path segments: `/api`, `/health`
- Parameter extraction: `/users/:id`, `/posts/:slug/comments/:commentId`
- Wildcard routing: `/static/*`, `/api/v1/*`
- Host-based routing: Route traffic based on domain names
- Path prefix stripping: Automatic removal of matched prefixes before forwarding

### Load Balancing

- **Round Robin**: Simple equal distribution
- **Least Connections**: Route to backend with fewest active connections
- **Weighted Round Robin**: Distribution based on backend capacity weights

### Resilience Features

- **Circuit Breaker**: Opens when backend failure rate exceeds threshold, preventing cascade failures
- **Health Checking**: Continuous monitoring with automatic backend reactivation
- **Timeout Management**: Configurable read, write, and idle timeouts
- **Graceful Shutdown**: Proper connection draining and cleanup

### Observability

- **Prometheus Metrics**: Request rates, latency percentiles, error rates, backend health
- **Structured Logging**: Configurable log levels with contextual information
- **Health Endpoints**: Service health checks and operational status
- **Circuit Breaker Events**: Detailed logging of circuit state transitions


## Architecture

### Design Philosophy

Balto follows a modular architecture where each component has a single responsibility:

- **Router**: Immutable routing tree for fast, thread-safe request matching
- **Proxy**: HTTP reverse proxy with streaming and context cancellation
- **Backend Pool**: Load balancing and circuit breaker management
- **Health Checker**: Active monitoring of backend service health
- **Service Manager**: REST API for dynamic service registration and management
- **Metrics Collector**: Prometheus metrics aggregation

### Data Flow

```
Client Request → Router → Load Balancer → Circuit Breaker → Backend Service
                      ↓
               Metrics Collection ← Health Monitoring
```

### External Control Plane Integration

Balto is designed to be managed by external systems:

- **Service Discovery**: Register/unregister services via REST API
- **Configuration Management**: Update routing rules and load balancing policies
- **Health Monitoring**: Receive health status updates from external monitors
- **Metrics Aggregation**: Export metrics for centralized monitoring systems

## Project Structure

```
├── cmd/balto/              # Main application entrypoint
├── internal/
│   ├── api/                # REST API for service management
│   │   ├── core/          # Service manager and types
│   │   ├── health/        # Health check endpoints
│   │   └── services/      # Service CRUD operations
│   ├── config/            # Configuration loading and validation
│   ├── core/
│   │   ├── backendpool/   # Backend management and load balancing
│   │   ├── balancer/      # Load balancing algorithms
│   │   ├── circuit/       # Circuit breaker implementation
│   │   └── core.go        # Core types and interfaces
│   ├── health/            # Health checking system
│   ├── metrics/           # Prometheus metrics collection
│   ├── proxy/             # HTTP reverse proxy
│   ├── router/            # Request routing engine
│   └── server/            # HTTP server with graceful shutdown
├── pkg/
│   ├── logger/            # Structured logging system
│   └── utils/             # Shared utilities
├── configs/               # Configuration files
├── deploy/                # Docker and deployment configurations
└── scripts/               # Development and build scripts
```


## Prerequisites

- **Go**: 1.23 or later
- **Make**: Build automation
- **Git**: Version control
- **Docker & Docker Compose**: For full observability stack (optional)

### Optional Development Tools

Install `golangci-lint` for local code linting:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc
source ~/.zshrc
```

## Quick Start

### Local Development

1. **Clone and build**:
   ```bash
   git clone https://github.com/diabeney/balto.git && cd balto
   make build
   ```

2. **Run the server**:
   ```bash
   make run
   ```

3. **Verify it's working**:
   ```bash
   curl -i http://localhost:5500/health
   curl -i http://localhost:5500/balto/health/stats
   ```

### Docker Development

For the full observability stack with Prometheus and Grafana:

```bash
docker compose up --build
```

Access points:
- **Balto**: http://localhost:5500
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

## Configuration

Balto uses YAML configuration for initial setup. Runtime configuration is managed through REST APIs by external control planes.

### Initial Configuration File

Create `configs/balto.config.yaml`:

```yaml
global:
  load_balancing:
    algorithm: round-robin
  logging:
    level: ["info", "warn", "error"]  # Multiple levels supported
    path: /var/log/balto/balto.log    # Optional file logging
  metrics:
    enabled: true
  timeouts:
    read: 5s
    write: 5s
    idle: 30s

server:
  port: ":5500"

services:
  - domain: api.example.com
    path_prefix: "/api/v1/*"
    ports: ["8081", "8082", "8083"]
  - domain: web.example.com
    path_prefix: "/*"
    ports: ["3000"]
```

### Configuration Options

#### Global Settings

- **load_balancing.algorithm**: Load balancing strategy
  - `round-robin`: Equal distribution
  - `least-connections`: Route to least loaded backend
  - `weighted-rr`: Weighted distribution based on backend capacity
- **logging.level**: Array of log levels to enable
  - `["debug", "info", "warn", "error"]` for all levels
  - `["info", "warn"]` for production logging
- **logging.path**: File path for logging (logs to stdout if not specified)
- **metrics.enabled**: Enable Prometheus metrics collection
- **timeouts**: HTTP timeout configurations

#### Service Configuration

- **domain**: Host header to match for routing
- **path_prefix**: URL path pattern with parameter and wildcard support
- **ports**: Backend service endpoints (ports or full URLs)

## REST API

Balto provides a REST API for external control planes to manage services dynamically.

### Base URL
All API endpoints are prefixed with `/balto`

### Service Management

#### List All Services
```http
GET /balto/services
```

Response:
```json
[
  {
    "id": "api.example.com-/api/v1/*",
    "domain": "api.example.com",
    "path_prefix": "/api/v1/*",
    "ports": ["8081", "8082"],
    "created_at": "2025-01-14T10:30:00Z",
    "updated_at": "2025-01-14T10:30:00Z"
  }
]
```

#### Register New Service
```http
POST /balto/services
Content-Type: application/json

{
  "domain": "api.example.com",
  "path_prefix": "/api/v1/*",
  "ports": ["8081", "8082", "8083"]
}
```

#### Get Specific Service
```http
GET /balto/services/{service_id}
```

#### Update Service
```http
PUT /balto/services/{service_id}
Content-Type: application/json

{
  "domain": "api.example.com",
  "path_prefix": "/api/v1/*",
  "ports": ["8081", "8084"]  // Updated ports
}
```

### Health and Monitoring

#### Service Health Stats
```http
GET /balto/health/stats
```

Returns operational statistics and health status.

#### Prometheus Metrics
```http
GET /balto/metrics
```

Exposes metrics in Prometheus format for monitoring systems.

### API Response Codes

- `200 OK`: Successful operation
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request data
- `404 Not Found`: Resource not found
- `405 Method Not Allowed`: HTTP method not supported
- `500 Internal Server Error`: Server error

## Circuit Breaker

Balto includes sophisticated circuit breaker functionality to prevent cascade failures.

### How It Works

1. **Closed State**: Normal operation, requests flow through
2. **Open State**: High failure rate detected, requests fail fast
3. **Half-Open State**: Testing if backend has recovered

### Configuration

Circuit breaker settings are configured per service during registration:

- **Failure Threshold**: Consecutive failures before opening (default: 10)
- **Success Threshold**: Consecutive successes to close breaker (default: 10)
- **Timeout**: How long to wait before half-open testing (default: 10s)
- **Max Half-Open Requests**: Concurrent test requests allowed (default: 5)

### Monitoring

Circuit breaker events are logged with structured information:

```
Circuit breaker opened backend_id=api-backend-8081 failure_threshold=10
Circuit breaker half-open (probing) backend_id=api-backend-8081 timeout_seconds=10
Circuit breaker closed backend_id=api-backend-8081 success_threshold=10
```

## Observability

Balto provides comprehensive monitoring capabilities through Prometheus metrics and structured logging.

### Metrics Collection

Balto exposes Prometheus metrics at `/balto/metrics`:

- **Request Metrics**: Rate, latency percentiles, error rates
- **Backend Health**: Per-backend request counts and health status
- **Circuit Breaker**: State transitions and failure counts
- **System Metrics**: CPU, memory, and Go runtime statistics

### Docker Compose Stack

For full observability setup with Prometheus and Grafana:

```bash
docker compose up --build
```

**Access Points:**
- **Balto**: http://localhost:5500
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

**Grafana Dashboards:**
- **Balto Application Metrics**: Request rates, latency, error rates
- **Per-Service Analytics**: Traffic distribution across services
- **Backend Performance**: Individual backend health and throughput
- **System Resources**: CPU, memory, and container metrics

### Docker Networking

For local development with Docker Compose:

- Set `BALTO_UPSTREAM_HOST=host.docker.internal` to reach host services
- Use container names for inter-container communication
- Configure service ports as full URLs for cross-network routing

## Development

### Testing

Run the complete test suite:

```bash
make test
```

### Code Quality

Format and lint code:

```bash
make format    # Format Go code
make lint      # Run linter
make lint-fix  # Auto-fix linting issues
```

### Git Hooks

Install pre-commit hooks for automatic code quality checks:

```bash
make install-hooks
```

The hooks will run `go fmt` and `golangci-lint` before each commit.

### Routing Engine

Balto's routing system supports flexible path matching:

**Path Patterns:**
- `/api` - Exact match
- `/users/:id` - Parameter extraction (creates `X-Param-id` header)
- `/static/*` - Wildcard matching
- `/api/v1/*` - Prefix with wildcard

**Routing Rules:**
- Routes are matched by domain first, then by path specificity
- Parameters are extracted and forwarded as HTTP headers
- Wildcard routes support prefix stripping before forwarding to backends
- No automatic fallback - routes must be explicitly defined

### Continuous Integration

GitHub Actions pipeline includes:
- Dependency management (`go mod tidy`)
- Code formatting verification (`gofmt`)
- Test execution (`go test ./...`)
- Linting (`golangci-lint`)

## Deployment

### Docker Compose

Balto provides Makefile commands for easy Docker operations:

```bash
# Start the full stack (Balto + Prometheus + Grafana)
make run

# Stop all services
make stop

# View logs
make show-logs                    # Show Balto logs (last 200 lines)
make show-logs target=prometheus  # Show Prometheus logs
make show-logs target=grafana     # Show Grafana logs
make show-logs tail=1000          # Show last 1000 lines
```

For manual Docker operations:

```bash
# Build image manually
docker build -f build/Dockerfile -t balto:latest .

# Run container manually
docker run -p 5500:5500 -v $(pwd)/configs:/app/configs balto:latest
```

### Kubernetes

Deploy to Kubernetes using the provided manifests in the `deploy/` directory.

### Configuration Management

For production deployments:

1. **Static Configuration**: Use YAML files for initial setup
2. **Dynamic Management**: Use REST APIs for runtime service registration
3. **Configuration Validation**: All changes are validated before application
4. **Hot Reload**: Zero-downtime configuration updates

## Operational Features

### Health Checking

Balto performs continuous health monitoring of backend services:

- **Active Checks**: Periodic HTTP/TCP health probes
- **Passive Checks**: Request failure analysis
- **Automatic Recovery**: Healthy backends are automatically reactivated
- **Configurable Thresholds**: Failure/success thresholds per service

### Circuit Breaker Integration

Circuit breakers work alongside health checking:

- **Failure Detection**: Opens circuit when failure rate exceeds threshold
- **Fast Fail**: Prevents requests to known unhealthy backends
- **Recovery Testing**: Half-open state for gradual recovery testing
- **Logging**: All state transitions are logged with context

### Load Balancing

Multiple algorithms available:

- **Round Robin**: Simple equal distribution
- **Least Connections**: Routes to least loaded backend
- **Weighted Round Robin**: Capacity-based distribution

### Metrics and Monitoring

Comprehensive observability:

- **Request Metrics**: Throughput, latency, error rates
- **Backend Metrics**: Per-backend performance and health
- **System Metrics**: Resource usage and performance
- **Custom Metrics**: Circuit breaker events, routing decisions


## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Write tests for new functionality
4. Ensure all tests pass: `make test`
5. Format and lint code: `make format && make lint-fix`
6. Commit changes: `git commit -am 'Add your feature'`
7. Push to branch: `git push origin feature/your-feature`
8. Submit a pull request

### Development Guidelines

- Follow Go best practices and effective Go guidelines
- Write comprehensive tests for new features
- Update documentation for API changes
- Ensure backward compatibility for configuration formats
- Use structured logging with appropriate log levels

## License

Licensed under Apache License 2.0. See `LICENSE` file for details.

## Support

- **Issues**: GitHub Issues for bug reports and feature requests
- **Documentation**: This README and inline code documentation
- **Community**: GitHub Discussions for questions and discussions

---

**Balto** - High-performance API gateway for modern microservices architectures.


