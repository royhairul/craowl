# Craowl

**Universal Data Acquisition Platform**

Craowl is a high-performance, modular data acquisition engine that collects structured data from multiple platforms using the optimal method automatically—whether through APIs, browser automation, GraphQL, WebSocket, or HTML parsing.

## Features

- **Multi-Method Acquisition**: API, browser automation, GraphQL, WebSocket, HTML parsing
- **Plugin Architecture**: Easily add new platforms without modifying core code
- **Three Interfaces**: CLI, REST API, and Go SDK
- **Smart Strategy Selection**: Automatically choose the fastest method
- **Session Management**: Persistent authentication with multi-account support
- **Export Formats**: JSON, CSV, Excel, PostgreSQL, Redis
- **Production Ready**: Rate limiting, retry logic, monitoring, scheduling

## Supported Platforms (v1.0)

- Shopee
- Instagram
- TikTok
- Tokopedia
- Facebook
- Generic (any website via HTML parsing)

## Quick Start

### Installation

```bash
go install github.com/yourusername/craowl/cmd/craowl@latest
```

### CLI Usage

```bash
# Login
craowl login shopee --cookie-file=cookies.json

# Crawl data
craowl crawl shopee --type=product --id=12345 --output=json

# Schedule job
craowl schedule --platform=shopee --cron="0 */6 * * *" --type=product --id=12345

# Batch crawl
craowl batch --platform=instagram --input=usernames.txt --output-dir=results/
```

### Go SDK

```go
package main

import (
    "context"
    "github.com/yourusername/craowl/pkg/craowl"
)

func main() {
    client := craowl.New(craowl.Config{})
    
    // Login
    session, _ := client.Login(context.Background(), craowl.LoginOptions{
        Platform: "shopee",
        Method:   craowl.MethodCookie,
        Cookies:  cookies,
    })
    
    // Crawl
    result, _ := client.Crawl(context.Background(), craowl.CrawlOptions{
        Platform: "shopee",
        Target: craowl.Target{
            Type: "product",
            ID:   "12345",
        },
        Session: session,
    })
}
```

### REST API

```bash
# Start server
craowl serve --port=8080

# Login
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"platform":"shopee","method":"cookie","cookies":[...]}'

# Crawl
curl -X POST http://localhost:8080/api/v1/crawl \
  -H "Content-Type: application/json" \
  -d '{"platform":"shopee","target":{"type":"product","id":"12345"}}'
```

## Architecture

```
craowl/
├── cmd/craowl/          # CLI application
├── internal/
│   ├── core/            # Domain logic
│   ├── acquisition/     # Acquisition methods (API, browser, etc.)
│   ├── auth/            # Authentication & session management
│   ├── storage/         # Storage adapters
│   └── api/             # REST API
├── pkg/craowl/          # Public SDK
└── plugins/             # Platform plugins
    ├── shopee/
    ├── instagram/
    └── ...
```

## Configuration

Create `~/.craowl/config.yaml`:

```yaml
log_level: info
concurrency:
  max_workers: 10
  browser_workers: 3
rate_limit:
  enabled: true
  requests_per_second: 10
storage:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    database: craowl
cache:
  type: redis
  redis:
    host: localhost
    port: 6379
```

## Development

### Prerequisites

- Go 1.19+
- PostgreSQL 14+
- Redis 7+
- Chrome/Chromium (for browser automation)

### Build

```bash
make build
```

### Test

```bash
make test
```

### Run Locally

```bash
go run cmd/craowl/main.go
```

## Adding a New Platform Plugin

1. Create plugin directory:
```bash
mkdir -p plugins/myplatform
```

2. Implement Platform interface:
```go
package myplatform

type MyPlatform struct {}

func (p *MyPlatform) Name() string { return "myplatform" }
func (p *MyPlatform) Supports(method AcquisitionMethod) bool { ... }
func (p *MyPlatform) Login(ctx context.Context, creds Credentials) (*Session, error) { ... }
func (p *MyPlatform) Crawl(ctx context.Context, target Target, opts CrawlOptions) (*Result, error) { ... }
func (p *MyPlatform) Extract(ctx context.Context, data []byte) (interface{}, error) { ... }
```

3. Register plugin:
```go
registry.Register(myplatform.New(config))
```

See [Plugin Development Guide](docs/plugin-development.md) for details.

## Performance

- **API Acquisition**: P95 <500ms, P99 <1s
- **Browser Acquisition**: P95 <8s, P99 <15s
- **Throughput**: >100 items/min (API), >20 items/min (browser)
- **Memory**: <50MB idle, <200MB per browser worker

## Documentation

- [Architecture](docs/architecture.md)
- [API Reference](docs/api-reference.md)
- [Plugin Development Guide](docs/plugin-development.md)
- [Product Requirements Document](PRD.md)

## Roadmap

- **v1.0** (8 months): Core engine, 5 platforms, CLI, API, SDK
- **v1.5** (12 months): Real-time streaming, web dashboard
- **v2.0** (18 months): Mobile reverse engineering, OCR, AI features

## Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

## License

[MIT License](LICENSE.md)

## Support

- Issues: [GitHub Issues](https://github.com/yourusername/craowl/issues)
- Documentation: [docs/](docs/)
- Email: support@craowl.dev

---

**Built with ❤️ using Go**
