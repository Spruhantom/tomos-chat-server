# TomChat Server

Self-hosted TomChat backend for Debian 13.

## Stage 3: Conversations and realtime encrypted-message transport

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
- Direct and group conversation creation (2–100 members)
- PostgreSQL message persistence with per-conversation sequence cursors and paginated history
- Authenticated WebSocket delivery with session-bound sockets, ping/pong, per-user connection caps and bounded queues
- Idempotent message submission using client-generated message IDs
- Opaque ciphertext and nonce storage; server does not decrypt message bodies

Not implemented yet: client-side cryptography and device-key distribution, admin web UI, account recovery, multi-instance WebSocket fan-out, and voice/video calls. Do not expose this service directly to the public internet until TLS, rate limiting, monitoring, and a security review are in place.

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

The server applies its idempotent schema on startup, including conversations, membership and encrypted message tables.

### Messaging payloads

Message requests contain a client-generated UUID, base64 ciphertext, base64 nonce, encryption version 1 and an optional key ID. The server stores the ciphertext as opaque bytes and never accepts a plaintext message field. Clients must encrypt before sending; the current client has not yet implemented the cryptographic protocol, so this is an E2EE-ready transport boundary rather than a complete E2EE system.

WebSocket events are documented in the TomChat Protocol repository. The current realtime hub is process-local and intended for one home-server instance. PostgreSQL is the source of truth; clients recover missed events by fetching message history. Do not horizontally scale this version without adding shared pub/sub.

## API

- `GET /healthz`
- `GET /api/v1/version`
- `GET /api/v1/registration`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/users/me` (Bearer token)
- `POST /api/v1/auth/logout` (Bearer token)
- `GET /api/v1/conversations` (Bearer token)
- `POST /api/v1/conversations` (Bearer token)
- `GET /api/v1/conversations/{conversationID}/messages?limit=50&before={sequence} or `after={sequence}` (mutually exclusive)` (Bearer token)
- `POST /api/v1/conversations/{conversationID}/messages` (Bearer token)
- `GET /api/v1/ws` (WebSocket upgrade with Bearer token)

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
