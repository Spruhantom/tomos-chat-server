# TomChat Server

Self-hosted TomChat backend for Debian 13.

## Stage 2: Authentication foundation

Implemented:
- PostgreSQL-backed user accounts and sessions
- Argon2id password hashing
- Public/private registration switch
- Login with random bearer tokens (stored as SHA-256 token hashes)
- Authenticated `GET /api/v1/users/me`
- Authenticated `POST /api/v1/auth/logout`
- Database-backed health check

Not implemented yet: message persistence, WebSocket delivery, admin web UI, account recovery, rate limiting, device keys, end-to-end encryption, and calling. Do not expose this service directly to the internet until TLS, rate limiting, monitoring, and a full security review are in place.

## Configuration

Set:
- `TOMCHAT_DATABASE_URL` — PostgreSQL connection string (required)
- `TOMCHAT_LISTEN_ADDR` — defaults to `127.0.0.1:8080`
- `TOMCHAT_REGISTRATION_ENABLED` — set to `true` to allow public registration; defaults to private registration

The server applies the initial idempotent schema on startup. For production, use a dedicated database role and a strong secret.

## Run

```bash
export TOMCHAT_DATABASE_URL='postgres://tomchat:password@127.0.0.1:5432/tomchat?sslmode=disable'
export TOMCHAT_REGISTRATION_ENABLED=false
go run ./cmd/tomchat-server
```

## API

- `GET /healthz`
- `GET /api/v1/version`
- `GET /api/v1/registration`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/users/me` (Bearer token)
- `POST /api/v1/auth/logout` (Bearer token)

Passwords must be 12–256 characters. Usernames are 3–32 ASCII letters, digits, dots, underscores, or hyphens.

## Debian 13

Install PostgreSQL 16 or later and Go 1.23+, configure the environment, then run the service. A hardened systemd unit is in `deployments/systemd`.

## Licence

GPL-3.0-or-later.
