# TomChat Server

Self-hosted TomChat backend for Debian 13.

## Stage 2: Authentication foundation

Implemented:
- PostgreSQL-backed accounts and sessions
- Argon2id password hashing
- Public/private registration switch
- First-run administrator bootstrap
- Login with random bearer tokens; only SHA-256 token hashes are stored
- Basic in-memory login rate limiting: 10 attempts per source IP per minute
- Bounded HTTP request/header timeouts
- Authenticated current-user endpoint and logout
- Database-backed health check

Not implemented yet: message persistence, WebSocket delivery, admin web UI, account recovery, rate limiting, device keys, end-to-end encryption, and voice/video calls. Do not expose this service directly to the public internet until TLS, rate limiting, monitoring, and a security review are in place.

## Configuration

Required:
- `TOMCHAT_DATABASE_URL` — PostgreSQL connection string

Optional:
- `TOMCHAT_LISTEN_ADDR` — defaults to `127.0.0.1:8080`
- `TOMCHAT_REGISTRATION_ENABLED` — set to `true` to allow public registration; defaults to private registration
- `TOMCHAT_BOOTSTRAP_ADMIN_USERNAME` and `TOMCHAT_BOOTSTRAP_ADMIN_PASSWORD` — together, create the first administrator only when the database has no users

The bootstrap credentials are ignored after the first account exists. Remove them from the service environment after successful first startup.

Login rate limiting is per-process and resets on restart. It uses the TCP peer address and deliberately does not trust `X-Forwarded-For`. If deploying behind a reverse proxy or running multiple server instances, configure rate limiting at the trusted proxy/load balancer as well.

## Run

```bash
export TOMCHAT_DATABASE_URL='postgres://tomchat:password@127.0.0.1:5432/tomchat?sslmode=disable'
export TOMCHAT_REGISTRATION_ENABLED=false
export TOMCHAT_BOOTSTRAP_ADMIN_USERNAME=admin
export TOMCHAT_BOOTSTRAP_ADMIN_PASSWORD='use-a-long-unique-password-here'
go run ./cmd/tomchat-server
```

The server applies its initial idempotent schema on startup.

## API

- `GET /healthz`
- `GET /api/v1/version`
- `GET /api/v1/registration`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/users/me` (Bearer token)
- `POST /api/v1/auth/logout` (Bearer token)

Passwords must be 12–256 characters. Usernames are 3–32 ASCII letters, digits, dots, underscores, or hyphens. Login sessions expire after 24 hours.

## Development

Requires Go 1.23+ and PostgreSQL 16+.

```bash
go mod download
go test ./...
go vet ./...
go run ./cmd/tomchat-server
```

## Licence

GPL-3.0-or-later.
