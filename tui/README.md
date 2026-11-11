# TomChat TUI

Rust/Ratatui terminal UI foundation for TomChat. This milestone provides keyboard navigation, a contacts panel, conversation view, composer and local peer-hint configuration.

**Not a functioning messenger yet:** the TUI does not currently authenticate to the Go server, connect a WebSocket, encrypt or send messages. Do not enter secrets or assume saved contacts are verified.

## Build on Arch Linux

```bash
sudo pacman -S --needed rust cargo pkgconf
cd tui
cargo build --release
cargo run --release
```

## Build on Debian 13

```bash
sudo apt update
sudo apt install -y cargo rustc pkg-config build-essential
cd tui
cargo build --release
cargo run --release
```

For a packaged .deb, see ../packaging/debian/README.md.

## Controls
- / enter command/message mode
- F1 help
- Tab switch panels
- Up/Down select contact
- Esc leave input mode
- q quit

Configuration is stored under XDG_CONFIG_HOME/tomchat/config.json (normally ~/.config/tomchat/config.json) with owner-only file permissions.

## Planned integration
The next milestone is authenticated API and WebSocket integration with the self-hosted TomChat Server, followed by client-side encryption. Server transport TLS is not a substitute for E2EE. The client must never submit plaintext message bodies.
