# TomChat Linux client roadmap

## Milestone 1 — terminal foundation
- [x] Rust/Ratatui application under `tui/`.
- [x] Keyboard-driven contacts and conversation layout.
- [x] Local peer-hint configuration with owner-only permissions.
- [x] Arch Linux build metadata.
- [ ] Validate formatting, compilation, tests and Clippy in CI; latest runs are queued.

## Milestone 2 — shared peer core
- [x] Create a shared Rust workspace crate for Noise XX sessions.
- [x] Keep JNI optional so Android and Linux use the same protocol implementation.
- [x] Migrate Android Noise JNI bridge and identity/contact-key binding source.
- [x] Link the Linux client to the shared Rust crate.
- [x] Add a direct TCP peer-channel adapter with bounded frames and a loopback handshake/message test.
- [x] Add Linux X25519 identity storage through the desktop Secret Service, with zeroization and fail-closed behaviour.
- [ ] Migrate Android contact and conversation UI integration and verify Android build/tests.

## Milestone 3 — direct connectivity
- [x] Direct TCP listener/connector API for known, reachable peers.
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
- [ ] Integrate Linux and Android composers with the encrypted peer channel.\n- [x] Define a versioned Android-compatible application envelope in Rust and Kotlin, with a shared golden test vector.\n- [x] Add encrypted delivery acknowledgements, persist outgoing messages as queued until ACK, and expose queued-message retry on Android.\n- [ ] Wire queued-message retry and receive-loop lifecycle into the Android UI; implement equivalent Linux durable outbox handling.
- [x] Add acknowledgement frames and Android encrypted queued-message persistence/retry API.\n- [ ] Finish cross-platform durable outbox, duplicate suppression and reconnect lifecycle.
- [ ] Run Android/Linux cross-platform interoperability tests and adversarial tests. Android direct Noise listener/connector has an instrumentation loopback test; this is not yet a physical Android-to-Linux test.

## Milestone 5 — calling and files
- [ ] Direct call signalling over an authenticated peer channel.
- [ ] Evaluate authenticated media encryption and direct media transport.
- [ ] Resumable encrypted file transfer.
- [ ] Physical-device battery, latency, loss and interruption tests.

## Validation notes
The direct TCP test currently uses loopback and generated ephemeral test identities. It proves that the Linux adapter can run the shared Noise handshake and exchange encrypted records; it does not prove that persistent identity storage, the user interface, cross-platform Android/Linux interoperability or remote-network reachability are complete.
