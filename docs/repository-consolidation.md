# Repository consolidation

This repository is the primary TomChat project following the move to a self-hosted server architecture.

## Imported scope
The planned consolidation brings across the TUI foundation (Rust/Ratatui source, Cargo manifest, Arch PKGBUILD and usage guide), relevant Android product/design documentation, and the TUI roadmap. Earlier zero-server architecture records are historical only and must not override the active server-backed architecture.

## Outstanding migration
The Android application source tree and its native Rust/JNI implementation have not yet been copied. They require a separate source migration and Android build validation. Imported Android design material must be reconciled with the server-backed direction before implementation resumes.

## Current limitations
- Client-side cryptography and device-key distribution are not complete.
- TUI is not yet connected to the server and does not send messages.
- Voice/video, account recovery, admin UI and multi-instance WebSocket fan-out are not implemented.
- Do not expose the service publicly until TLS, production rate limiting, monitoring and security review are complete.

Source project: https://github.com/Spruhantom/tomos-chat-cliant
