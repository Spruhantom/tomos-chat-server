# TomChat TUI

Rust/Ratatui terminal UI foundation for TomChat. It provides keyboard navigation, a contacts panel, conversation view, composer and local peer-hint configuration.

**Not a functioning messenger yet:** this build does not discover peers, establish direct connections, encrypt or deliver messages. Do not enter secrets or assume saved contacts are verified.

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
``

## Controls
- / enter command/message mode
- F1 help
- Tab switch panels
- Up/Down select contact
- Esc leave input mode
- q quit

Configuration is stored under XDG_CONFIG_HOME/tomchat/config.json (normally ~/.config/tomchat/config.json) with owner-only file permissions.

## Planned integration
The next milestone is a shared Rust peer core for identity, discovery, direct transport, local encrypted storage and reviewed end-to-end cryptography. Network delivery must remain peer-to-peer; the application must clearly report when a peer cannot be reached.
