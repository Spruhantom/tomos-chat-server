# TomChat Server

Self-hosted TomChat backend for **Debian 13**.

## Current status

This repository is the initial development scaffold. It currently provides a minimal Go HTTP service with health and version endpoints. Authentication, persistence, messaging, administration, encryption, and calls are not implemented yet.

## Development

Requires Go 1.23 or later.

```bash
go test ./...
go run ./cmd/tomchat-server
```

The development server binds to `127.0.0.1:8080` by default. Set `TOMCHAT_LISTEN_ADDR` to override it.

- `GET /healthz` — health check
- `GET /api/v1/version` — server version

## Architecture

- Go HTTP API
- PostgreSQL planned for persistence
- WebSockets planned for realtime events
- React + TypeScript planned for the admin dashboard
- systemd deployment planned for Debian 13

## Security notice

Do not expose this scaffold to the public internet as a production chat service. It has no authentication or database yet. Use TLS at the reverse proxy when network deployment is introduced.

## Licence

GPL-3.0-or-later. See [LICENSE](LICENSE).
