# TomChat Linux client roadmap

## Milestone 1 — terminal foundation
- [x] Rust/Ratatui application under `tui/`.
- [x] Keyboard-driven contacts and conversation layout.
- [x] Local peer-hint configuration with owner-only permissions.
- [x] Arch Linux build metadata.
- [ ] Validate formatting, compilation, tests and Clippy in CI.

## Milestone 2 — shared peer core
- [x] Shared Rust workspace crate for Noise XX sessions.
- [x] Optional JNI feature so Android and Linux use the same protocol implementation.
- [x] Android Noise JNI bridge and identity/contact-key binding source migrated.
- [x] Linux client linked to the shared Rust crate.
- [x] Direct TCP peer-channel adapter with bounded frames and loopback handshake/message tests.
- [x] Linux X25519 identity storage through desktop Secret Service, with zeroization and fail-closed behaviour.
- [x] Rust and Kotlin versioned application envelope with a shared golden wire-format vector.
- [x] Android direct Noise-pinned TCP listener/connector with encrypted loopback instrumentation test.
- [ ] Wire Linux TUI contact management and UI to the direct peer channel.
- [ ] Run physical Android-to-Linux interoperability tests.

## Milestone 3 — direct connectivity
- [x] Direct TCP listener/connector APIs for known, reachable peers.
- [x] Noise XX authentication pins the peer static key before enabling encrypted records.
- [ ] Bluetooth LE discovery and GATT proof of concept.
- [ ] Bluetooth Classic RFCOMM where supported.
- [ ] Wi-Fi LAN discovery using mDNS/local multicast.
- [ ] Direct QUIC/IP for mutually reachable remote devices.
- [ ] Explicit unreachable status when direct paths fail; no relay fallback.

## Milestone 4 — secure messaging
- [ ] Review and finalise pairwise E2EE threat model and implementation.
- [ ] Evaluate OpenMLS for group messaging.
- [ ] Persist keys and ratchet state securely on Linux and Android.
- [x] Android encrypted message history and queued outgoing-message persistence.
- [x] Encrypted ACK frames; Android marks outgoing messages sent only after peer acknowledgement.
- [x] Android queued-message retry API and receive-side duplicate replacement by stable message ID.
- [ ] Wire Android retry/receive lifecycle into the Compose UI.
- [ ] Implement equivalent durable Linux outbox, ACK processing and reconnect retry.
- [ ] Complete cross-platform interoperability and adversarial tests.

## Milestone 5 — calling and files
- [ ] Direct call signalling over an authenticated peer channel.
- [ ] Evaluate authenticated media encryption and direct media transport.
- [ ] Resumable encrypted file transfer.
- [ ] Physical-device battery, latency, loss and interruption tests.

## Validation notes
Loopback tests verify the Noise handshake and encrypted records on each platform. The shared golden vector verifies the application-envelope wire format. These do not yet prove physical Android-to-Linux interoperability, user-interface integration, remote-network reachability or production security. The cryptographic prototype still requires independent review.
