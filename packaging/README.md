# Linux installation

TomChat is a decentralised peer-to-peer application. Linux builds install the local client only; no central infrastructure is required or installed.

## Debian 13 online installer

Run as your normal user:

    curl -fsSL https://raw.githubusercontent.com/Spruhantom/tomos-chat-server/migration/tomchat-consolidation/install-debian.sh | bash

The installer checks the distribution, installs Rust build dependencies, builds the terminal client from source and installs `/usr/local/bin/tomchat`. Review the script before running it.

The installer currently targets Debian 13 (Trixie) and the migration branch. Once this work is merged, the default branch in the command will be updated to `main`.

## Build manually on Debian

    sudo apt update
    sudo apt install -y cargo rustc pkg-config build-essential
    cd tui
    cargo build --release
    cargo test

## Build on Arch Linux

    sudo pacman -S --needed rust cargo pkgconf
    cd tui
    cargo build --release
    cargo test

The current terminal application is a UI foundation. It does not yet discover peers, establish direct connections, encrypt or deliver messages.
