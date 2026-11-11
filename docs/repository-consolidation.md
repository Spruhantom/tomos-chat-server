# Repository consolidation

This repository is being realigned around TomChat's decentralised peer-to-peer product direction.

## Active scope
- Rust shared protocol and networking core.
- Android client using Kotlin/Jetpack Compose and a narrow Rust FFI.
- Linux terminal client using Rust/Ratatui.
- Local-first identity, encrypted storage and outbox.
- Direct Bluetooth, Wi-Fi LAN and reachable direct-IP transports.
- Signed contact cards and user-controlled peer discovery.

## Removed from the active project
The central Go/PostgreSQL messaging backend, account API, hosted message persistence, deployment manifests, service units and centralised package workflow have been removed from this development branch. The project must not reintroduce central dependencies as a shortcut for discovery, delivery, notifications or media traversal.

## Migration status
The current repository contains the TUI foundation and architecture documentation. The Android application and its native Rust/JNI work from the former client project still need to be reviewed and migrated into this repository. The TUI is not yet a functioning messenger.

## Next work
1. Inspect and migrate the existing Android identity/contact-card and direct-peer prototype.
2. Extract a shared Rust core with a documented FFI contract.
3. Implement and test authenticated direct transport.
4. Integrate a reviewed pairwise E2EE protocol and persistent encrypted outbox.
5. Validate Bluetooth and Wi-Fi on two physical devices before expanding remote direct-IP support.

Original client source: https://github.com/Spruhantom/tomos-chat-cliant
