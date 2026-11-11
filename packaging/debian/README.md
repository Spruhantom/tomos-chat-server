# Debian packaging

TomChat supports Debian 13 (Trixie) as the initial Debian target. These scripts create local binary .deb packages from the source checkout; they are intended for development/testing and are not yet signed or published in an APT repository.

## Build packages

Install build tools:

    sudo apt update
    sudo apt install -y build-essential ca-certificates golang-go cargo rustc dpkg-dev

From the repository root:

    ./packaging/debian/build-packages.sh

The script produces:
- build/deb/tomchat-server_VERSION_amd64.deb
- build/deb/tomchat-tui_VERSION_amd64.deb

It currently targets amd64. Rust and Go dependencies are downloaded by their respective build tools. For production Debian source packages, dependency vendoring, reproducible builds, signing and APT repository publication still need to be added.

## Install the server

    sudo apt install ./build/deb/tomchat-server_*.deb
    sudoedit /etc/tomchat/tomchat-server.env
    sudo systemctl enable --now tomchat-server

Set TOMCHAT_DATABASE_URL to a PostgreSQL connection string. Keep registration disabled unless you intentionally want public registration. Bootstrap credentials should be used only for first administrator creation and removed from the environment afterwards.

The service binds to localhost by default. Put it behind a correctly configured TLS reverse proxy before remote access. The package does not install PostgreSQL or configure a database automatically.

## Install the TUI

    sudo apt install ./build/deb/tomchat-tui_*.deb
    tomchat

The TUI remains a UI foundation and is not yet connected to the server.
