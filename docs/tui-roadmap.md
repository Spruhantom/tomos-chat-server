# TomChat TUI roadmap

## Milestone 1 — terminal foundation
- [x] Rust/Ratatui application under `tui/`.
- [x] Keyboard-driven contacts and conversation layout.
- [x] Local peer-hint configuration with owner-only permissions.
- [x] Arch Linux PKGBUILD and build instructions.
- [ ] Validate formatting, compilation, tests and Clippy in CI.

## Milestone 2 — server connectivity
- [ ] Configure a user-selected TomChat Server URL.
- [ ] Implement registration/login and secure token storage.
- [ ] Add authenticated HTTP client and WebSocket connection.
- [ ] Recover history using server sequence cursors.
- [ ] Add reconnect handling and clear connection status.

## Milestone 3 — client-side encryption
- [ ] Define and review device identity and key enrolment.
- [ ] Implement a reviewed E2EE protocol and versioned ciphertext envelope.
- [ ] Ensure plaintext is never sent to or stored by the server.
- [ ] Add cross-platform interoperability and adversarial tests.

## Milestone 4 — features
- [ ] Direct and group messaging.
- [ ] File transfer and voice signalling/media after security and transport review.
- [ ] Account/device recovery and revocation.
