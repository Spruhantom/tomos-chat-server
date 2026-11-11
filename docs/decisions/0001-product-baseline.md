# Decision 0001: TomChat product baseline

- Status: accepted.
- Product: decentralised peer-to-peer messaging application.
- Network: direct device-to-device connections only.
- Discovery: Bluetooth LE, Bluetooth Classic where supported, local Wi-Fi discovery, and user-exchanged signed contact cards.
- No central accounts, public directory, relay, rendezvous, push gateway or hosted message storage.
- Identity and message keys are generated and controlled locally.
- Messages must be end-to-end encrypted and authenticated before transmission.
- Clients: Android and Linux terminal initially, sharing a Rust core.
- Delivery: encrypted local outbox; no guaranteed delivery while recipients are unreachable.

The previous centralised backend direction is rejected. This decision governs all new architecture and implementation work.
