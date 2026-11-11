# TomChat architecture

## Product direction
TomChat is a self-hosted messaging platform. Users and communities operate their own server instances; clients connect to a server selected by the user or administrator. There is no single mandatory public TomChat service.

The Go server manages accounts, sessions, conversation membership, durable ciphertext storage, history pagination and authenticated realtime delivery. It must never receive plaintext message content. Client-side end-to-end cryptography and device-key distribution remain incomplete until implemented and independently reviewed.

## Server components
- Go HTTP API and authenticated WebSocket endpoint.
- PostgreSQL durable source of truth for accounts, sessions, conversations, membership and opaque encrypted messages.
- Argon2id password hashing and hashed bearer-token storage.
- Per-process login throttling, bounded request timeouts and bounded realtime queues.
- Client-generated message IDs for idempotent submission.
- Sequence cursors and paginated history for reconnect recovery.

The realtime hub is process-local; the supported deployment is a single server instance. Multi-instance fan-out requires shared pub/sub and testing.

## Client responsibilities
Android, terminal and desktop clients must authenticate to a user-configured server over TLS in production; encrypt content before submission; verify recipient/device identity; manage keys locally; decrypt only on authorised devices; protect private keys with platform secure storage; never log plaintext, credentials or key material; and recover missed events from paginated history.

TLS protects client-server transport but is not E2EE. Do not describe the current transport boundary as complete end-to-end encryption.

## Message envelope
The API accepts client-generated message UUIDs, base64 ciphertext, base64 nonce, encryption version and optional key ID. The server stores ciphertext as opaque bytes and does not accept a plaintext body field. The cryptographic envelope, key establishment, rotation, device enrolment and group-key protocol still require implementation and review.

## Deployment
Default to private registration. Remove bootstrap credentials after first admin creation. Configure TLS before exposing beyond localhost. Add production-grade edge rate limiting, monitoring, backups, restore tests and security review. Do not horizontally scale the current WebSocket hub without shared fan-out.

## TUI status
The Rust/Ratatui app under `tui/` is currently a UI and local peer-hint foundation only. It does not yet authenticate to the Go API, connect a WebSocket, encrypt or send messages. Its next milestone is API authentication and client-side encryption integration.
