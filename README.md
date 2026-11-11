# TomChat

Self-hosted TomChat messaging platform. The Go server provides accounts, conversations, opaque encrypted-message storage and authenticated realtime delivery. Clients connect to a server selected by the user or community.

**Security status:** client-side cryptography, device-key distribution and a complete independently reviewed E2EE protocol are not yet implemented. The server stores ciphertext and does not accept plaintext message bodies; this transport boundary is not yet a complete E2EE system.

## Supported Linux platforms

- Debian 13 (Trixie): server and TUI binary packages can be built locally using the scripts in packaging/debian/.
- Arch Linux: TUI PKGBUILD is under tui/PKGBUILD.
- Other Linux distributions may work, but are not yet part of the tested support matrix.

## Repository layout
- cmd/tomchat-server — Go server executable.
- internal/ — server implementation.
- migrations/ — PostgreSQL schema.
- tui/ — Rust/Ratatui terminal client foundation (not yet connected to the server).
- packaging/debian/ — Debian package builder and systemd service.
- docs/architecture.md — active server-backed architecture.
- docs/repository-consolidation.md — migration status and outstanding work.
- docs/history/ — historical zero-server design records retained for reference.

## Server capabilities
PostgreSQL accounts/sessions; Argon2id password hashing; hashed bearer-token storage; private/public registration; first-run admin bootstrap; authenticated HTTP API; direct/group conversations; opaque ciphertext persistence; sequence cursors and paginated history; authenticated WebSocket delivery; idempotent message submission.

Not yet implemented: client cryptography/device-key distribution, account recovery, admin UI, multi-instance WebSocket fan-out, and voice/video calling.

## Run
Requires Go 1.23+ and PostgreSQL 16+.
```bash
export TOMCHAT_DATABASE_URL='postgres://tomchat:password@127.0.0.1:5432/tomchat?sslmode=disable'
export TOMCHAT_REGISTRATION_ENABLED=false
export TOMCHAT_BOOTSTRAP_ADMIN_USERNAME=admin
export TOMCHAT_BOOTSTRAP_ADMIN_PASSWORD='use-a-long-unique-password-here'
go run ./cmd/tomchat-server
```
Remove bootstrap credentials after the first administrator is created. Do not expose publicly until TLS, production rate limiting, monitoring, backup/restore and security review are in place.

## Debian 13

On Debian 13 (amd64), you can use the online installer (currently targeting the migration branch while it is under review):
```bash
curl -fsSL https://raw.githubusercontent.com/Spruhantom/tomos-chat-server/migration/tomchat-consolidation/install-debian.sh | bash
```
Review the script before running it. For manual local .deb builds, see [Debian packaging instructions](packaging/debian/README.md). Build local .deb packages with:
```bash
bash packaging/debian/build-packages.sh
```

## Arch Linux TUI
```bash
sudo pacman -S --needed rust cargo pkgconf
cd tui && cargo build --release && cargo test
```
The TUI is currently a UI foundation only; it does not yet authenticate, connect or send messages. See [tui/README.md](tui/README.md).

## API
- GET /healthz, GET /api/v1/version, GET /api/v1/registration
- POST /api/v1/auth/register, POST /api/v1/auth/login, POST /api/v1/auth/logout
- GET /api/v1/users/me
- GET/POST /api/v1/conversations
- GET/POST /api/v1/conversations/{conversationID}/messages
- GET /api/v1/ws (authenticated WebSocket)

## Tests
```bash
go mod download
go test ./...
go vet ./...
```

## Migration
Documentation and the TUI foundation from the former Android-first repository are being consolidated here. Android source and native Rust/JNI code have not yet been migrated. See [docs/repository-consolidation.md](docs/repository-consolidation.md).

GPL-3.0-or-later.
