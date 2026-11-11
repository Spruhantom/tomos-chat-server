# TomChat peer-to-peer architecture

## Product direction
TomChat is a decentralised application. Each installation owns its identity and local data. Devices communicate directly with other devices; there is no central account authority, message store, relay, rendezvous service, push gateway or public peer directory.

## Client architecture
- Android: Kotlin/Jetpack Compose owns UI, lifecycle, permissions, notifications and audio routing.
- Shared core: Rust owns peer identity, protocol state, discovery, transport selection, encrypted message envelopes, synchronisation and group-state integration.
- Linux: Rust/Ratatui terminal client uses the shared core rather than a separate protocol implementation.
- FFI: keep the Kotlin/Rust boundary small, typed, versioned and asynchronous.

## Direct discovery
- Bluetooth LE advertisements and GATT for nearby discovery and bounded data exchange.
- Bluetooth Classic RFCOMM where device support and pairing permit it.
- mDNS/local multicast for devices on the same Wi-Fi network.
- QR, NFC or user-shared signed contact cards for out-of-band identity and connection hints.
- Existing trusted peers may exchange signed reachability updates directly.
- No global stranger search is promised. Discovery is limited to nearby devices, known contacts and information users exchange themselves.

Use rotating privacy-preserving discovery identifiers. Do not broadcast stable identity keys, names, phone numbers or location.

## Direct transport
Select a direct path based on reachability and measured conditions:
1. Bluetooth LE for nearby discovery and small control payloads.
2. Bluetooth Classic for nearby data transfer where supported.
3. Wi-Fi LAN for higher-throughput local transfers.
4. Direct QUIC/IP for remote peers when both endpoints are reachable.

The application must never introduce a relay, rendezvous, proxy, hosted directory or fallback path. Carrier-grade NAT and restrictive firewalls mean some remote peers cannot connect. Report unreachable peers clearly. Do not imply guaranteed delivery while a recipient is offline.

Bluetooth data transports are not assumed to provide reliable low-latency voice. Initial calls require a separately validated direct IP/Wi-Fi media path and real-device testing.

## Identity and cryptography
Use maintained, reviewed cryptographic implementations; do not invent primitives or hand-roll a handshake. Device identity keys are generated locally and protected by platform secure storage. Contact cards are signed and include a fingerprint; self-signatures do not prove a real-world identity, so users must compare fingerprints through a separate trusted channel.

Use a reviewed pairwise protocol and evaluate OpenMLS for groups. Bind device enrolment, key changes, revocation, group membership and backup recovery to explicit threat-model requirements. Encrypt before transport. No peer, including a forwarding device, may learn plaintext intended for another peer.

## Messaging and synchronisation
- Durable encrypted local outbox with stable message IDs.
- Authenticated envelopes, replay protection, acknowledgements and per-conversation ordering.
- Direct peer-to-peer history and state synchronisation when devices are reachable.
- Pending messages remain encrypted on the sender's device and retry when the recipient becomes reachable.
- No guaranteed mailbox exists while all recipient devices are offline.
- Group membership changes are authenticated and auditable.
- Explicit encrypted replication to trusted devices may be added; it must remain user-controlled.

## Voice and files
Voice signalling travels over an already authenticated peer channel or is exchanged out of band. Evaluate an audited media stack and authenticated media encryption; do not add network traversal services. Group voice begins as a direct peer mesh with conservative participant limits. File transfer uses resumable encrypted chunks and authenticated manifests.

## Privacy and safety
- No plaintext in logs, crash reports or notifications by default.
- Validate and bound every peer-controlled input; defend against spam, replay and resource exhaustion.
- Provide block/report controls, contact approval and key-change warnings.
- Explain that losing every device and the recovery secret can permanently lose identity and history.
- Respect Android background limits; incoming alerts cannot be guaranteed when a device is offline, force-stopped or unreachable.

## Implementation status
The Linux TUI remains a local UI and contact-hint foundation and is not yet wired to its peer adapter. The shared Rust core exposes direct Noise XX TCP channels and an Android-compatible application envelope; Android has a matching Noise-pinned direct TCP listener/connector, encrypted message persistence, ACK handling and a queued-retry API. These are protocol/transport components, not a complete end-user messaging flow. The Android Compose UI, Linux TUI, reconnect lifecycle and durable Linux outbox are still outstanding. Loopback tests and golden vectors do not replace physical Android-to-Linux interoperability tests or adversarial security review.
