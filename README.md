# TomChat

TomChat is a decentralised, peer-to-peer messaging application. Devices communicate directly; there is no central account authority, message host, relay network, public directory or mandatory online service.

The application is being developed around a shared Rust core, with Android and Linux clients. Nearby discovery and direct communication are intended to use Bluetooth LE, Bluetooth Classic where supported, and Wi-Fi LAN. Remote communication uses direct IP/QUIC where both devices are reachable. QR/contact-card exchange provides an out-of-band way to establish peer identity and connection hints.

**Important network limitation:** many mobile networks and home routers use carrier-grade NAT or restrictive firewalls. Without relays or rendezvous infrastructure, some remote peers will be unreachable. TomChat must report that honestly rather than silently route through central infrastructure.

**Current implementation status:** the Rust/Ratatui terminal application is a user-interface foundation with local contact hints. It does not yet establish peer connections, encrypt or deliver messages. End-to-end cryptography, device-key management, reliable offline delivery and calling must not be described as complete until implemented and tested.

## Repository layout
- `tui/` — Rust/Ratatui Linux terminal client foundation.
- `docs/architecture.md` — active peer-to-peer architecture and transport constraints.
- `docs/tui-roadmap.md` — Linux client development milestones.
- `docs/account-age-safety.md` — safety requirements.

## Linux development

### Debian 13
```bash
curl -fsSL https://raw.githubusercontent.com/Spruhantom/tomos-chat-server/migration/tomchat-consolidation/install-debian.sh | bash
```
The installer builds and installs the current terminal UI from source. Review it before running.

### Arch Linux
```bash
sudo pacman -S --needed rust cargo pkgconf
cd tui
cargo build --release
cargo test
``

## Current limitations
- The TUI does not yet communicate with peers.
- Peer discovery, authenticated direct transport and message synchronisation remain to be implemented.
- The cryptographic protocol and device-key lifecycle require implementation, independent review and adversarial testing.
- Voice/video, file transfer, multi-device replication and background delivery are not yet implemented.

## Project principles
- No central infrastructure or mandatory third-party network service.
- Peer identity is cryptographic, not a display name or IP address.
- Encrypt and authenticate content end-to-end before any network transmission.
- Keep keys and message history under the user's control.
- Make reachability, trust and delivery status explicit.
- Preserve privacy by default and minimise metadata.

GPL-3.0-or-later.
