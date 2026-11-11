# Decision 0001: TomChat product baseline

- Status: accepted for the consolidated repository.
- Product direction: self-hosted server architecture.
- Server: Go HTTP/WebSocket service with PostgreSQL persistence.
- Clients: Android and terminal clients connect to a user-selected server.
- Message privacy: server stores opaque ciphertext; clients own end-to-end encryption.
- Security status: server-side encrypted-message transport exists, but client cryptography and device-key distribution remain incomplete.
- Distribution: server and clients are independently buildable; Arch TUI package metadata is under `tui/PKGBUILD`.

This decision supersedes the earlier zero-server architecture direction. Historical material is retained under `docs/history/` where imported.
