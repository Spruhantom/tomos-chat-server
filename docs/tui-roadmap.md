# TomChat Linux client roadmap

## Milestone 1 — terminal foundation
- [x] Rust/Ratatui application under `tui/`.
- [x] Keyboard-driven contacts and conversation layout.
- [x] Local peer-hint configuration with owner-only permissions.
- [x] Arch Linux build metadata.
- [ ] Validate formatting, compilation, tests and Clippy in CI.

## Milestone 2 — shared peer core
- [x] Create a shared Rust workspace crate for Noise XX sessions.
- [x] Keep JNI optional so Android and Linux use the same protocol implementation.
- [x] Migrate Android Noise JNI bridge and identity/contact-key binding source.
- [x] Link the Linux client to the shared Rust crate.
- [x] Add a direct TCP peer-channel adapter with bounded frames and a loopback handshake/message test.
- [ ] Add Linux secure identity storage through the OS credential service.
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
- [ ] Integrate Linux and Android composers with the encrypted peer channel.
- [ ] Implement acknowledgements, encrypted outbox retry and duplicate suppression.
- [ ] Add cross-platform interoperability and adversarial tests.

## Milestone 5 — calling and files
- [ ] Direct call signalling over an authenticated peer channel.
- [ ] Evaluate authenticated media encryption and direct media transport.
- [ ] Resumable encrypted file transfer.
- [ ] Physical-device battery, latency, loss and interruption tests.

## Validation notes
The direct TCP test currently uses loopback and generated ephemeral test identities. It proves that the Linux adapter can run the shared Noise handshake and exchange encrypted records; it does not prove that persistent identity storage, the user interface, cross-platform Android/Linux interoperability or remote-network reachability are complete.
