# TomChat Linux client roadmap

## Milestone 1 — terminal foundation
- [x] Rust/Ratatui application under `tui/`.
- [x] Keyboard-driven contacts and conversation layout.
- [x] Local peer-hint configuration with owner-only permissions.
- [x] Arch Linux build metadata.
- [ ] Validate formatting, compilation, tests and Clippy in CI.

## Milestone 2 — shared peer core
- [ ] Define Rust workspace crates for identity, contacts, transport, message envelopes and local storage.
- [ ] Define versioned interfaces for Android FFI and Linux.
- [ ] Add local encrypted database and durable encrypted outbox.
- [ ] Add identity/contact-card import and fingerprint verification.

## Milestone 3 — direct connectivity
- [ ] Bluetooth LE discovery and GATT proof of concept.
- [ ] Bluetooth Classic RFCOMM where supported.
- [ ] Wi-Fi LAN discovery using mDNS/local multicast.
- [ ] Authenticated direct peer handshake with replay protection.
- [ ] Direct QUIC/IP for mutually reachable remote devices.
- [ ] Explicit unreachable status when direct paths fail; no relay fallback.

## Milestone 4 — secure messaging
- [ ] Select and integrate reviewed pairwise E2EE protocol.
- [ ] Evaluate OpenMLS for group messaging.
- [ ] Persist keys and ratchet state securely.
- [ ] Implement encrypted message send/receive, acknowledgements and retry.
- [ ] Add cross-platform interoperability and adversarial tests.

## Milestone 5 — calling and files
- [ ] Direct call signalling over an authenticated peer channel.
- [ ] Evaluate authenticated media encryption and direct media transport.
- [ ] Resumable encrypted file transfer.
- [ ] Physical-device battery, latency, loss and interruption tests.
