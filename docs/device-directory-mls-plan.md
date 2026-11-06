# Device directory and MLS relay: server implementation plan

Status: design only. No device-key endpoints, MLS validation, or E2EE guarantees are implemented.

## Server trust boundary

The server is an untrusted MLS delivery service. It stores public device-directory records and opaque MLS protocol messages. It must never receive, generate, store or recover private device keys, group secrets, plaintext, or recovery secrets. TLS protects credentials and metadata in transit but is not E2EE.

The server must not claim that a device credential or MLS message is valid merely because it is well-formed base64. Cryptographic credential validation, group membership state transitions, epoch consistency and message decryption are client responsibilities.

## Planned PostgreSQL records

### devices
- device_id UUID primary key
- user_id UUID foreign key
- credential BYTEA (public MLS credential only)
- credential_fingerprint BYTEA
- display_name bounded text
- protocol_version integer
- status enum: pending, verified, revoked
- created_at, approved_at, revoked_at timestamps
- approval_proof BYTEA (public signature/proof only)

Unique constraints must prevent a device ID from being rebound to another user. Revoked device IDs cannot be reactivated.

### key_packages
- package_id UUID primary key
- device_id UUID foreign key
- package BYTEA (opaque signed public KeyPackage)
- expires_at timestamp
- claimed_at timestamp nullable
- claimed_by_device_id UUID nullable

Use a transaction with row locking or an equivalent atomic conditional update to guarantee single-use claims. Expired and claimed packages must not be returned.

### mls_messages
- id UUID primary key
- conversation_id UUID foreign key
- sender_device_id UUID foreign key
- client_message_id UUID
- server_sequence BIGINT per conversation
- message_type constrained enum
- protocol_version integer
- epoch BIGINT as untrusted routing metadata
- content BYTEA
- created_at timestamp

Unique (sender_device_id, client_message_id). Preserve server-assigned monotonic sequence and existing cursor semantics.

## API and authorisation

Implement the proposed endpoints in `device-directory.md` only after the protocol contract is final. Every route must derive user identity from the bearer session. Conversation-scoped directory reads must first verify conversation membership. MLS message writes must require an active, verified device that is an authorised member of that conversation. Revoked/pending devices must fail closed.

## Operational controls

Apply strict request and response size limits, per-account/device rate limits, bounded KeyPackage inventory, expiry and cleanup jobs, audit events for approval/revocation, and generic error responses that do not leak account enumeration. Avoid logging credential blobs, KeyPackages or MLS message bodies. Add migration rollback guidance and retention policy.

## Required tests

- unauthorised user cannot enumerate another user's devices;
- non-member cannot enumerate conversation devices or post MLS messages;
- pending and revoked devices cannot send or claim KeyPackages;
- approval signature is bound to the exact new device credential and nonce;
- KeyPackage claims are atomic under concurrent requests and single-use;
- idempotent MLS-message retry returns the original server record;
- message sequence and after-cursor pagination remain correct;
- size limits, expiry, malformed base64 and rate limits fail safely;
- revocation blocks subsequent requests immediately;
- transaction failures do not leave partial device or KeyPackage state.

## Rollout

Ship additive migrations and APIs behind a server feature flag. Keep existing REST/WebSocket messaging available for compatibility but do not let the client composer send plaintext. Enable MLS relay only after the client implementation and protocol tests are ready. Production E2EE claims require independent review of the complete client/server integration.
