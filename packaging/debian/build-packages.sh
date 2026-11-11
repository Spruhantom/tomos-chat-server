#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
VERSION="${VERSION:-0.1.0}"
ARCH="${DEB_ARCH:-amd64}"
OUT="$ROOT/build/deb"
WORK="$ROOT/build/debian-work"

command -v go >/dev/null || { echo "Missing Go toolchain" >&2; exit 1; }
command -v cargo >/dev/null || { echo "Missing Cargo toolchain" >&2; exit 1; }
command -v dpkg-deb >/dev/null || { echo "Missing dpkg-deb (install dpkg-dev)" >&2; exit 1; }

rm -rf "$WORK"
mkdir -p "$OUT" "$WORK/server/DEBIAN" "$WORK/server/usr/bin" "$WORK/server/etc/tomchat" "$WORK/server/lib/systemd/system"
mkdir -p "$WORK/tui/DEBIAN" "$WORK/tui/usr/bin" "$WORK/tui/usr/share/doc/tomchat-tui"

(
  cd "$ROOT"
  go build -trimpath -ldflags="-s -w" -o "$WORK/server/usr/bin/tomchat-server" ./cmd/tomchat-server
)
(
  cd "$ROOT/tui"
  cargo build --release
  install -m 0755 target/release/tomchat "$WORK/tui/usr/bin/tomchat"
  install -m 0644 README.md "$WORK/tui/usr/share/doc/tomchat-tui/README.md"
)

cat > "$WORK/server/DEBIAN/control" <<CONTROL
Package: tomchat-server
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Maintainer: TomChat Project <maintainers@tomchat.invalid>
Description: Self-hosted TomChat messaging server
 Go HTTP and WebSocket server with PostgreSQL-backed accounts,
 conversations and opaque encrypted-message transport.
Depends: libc6
CONTROL

cat > "$WORK/tui/DEBIAN/control" <<CONTROL
Package: tomchat-tui
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Maintainer: TomChat Project <maintainers@tomchat.invalid>
Description: TomChat terminal client foundation
 Rust and Ratatui terminal interface for the TomChat platform.
 This release is a UI foundation and does not yet send messages.
Depends: ${shlibs:Depends}, ${misc:Depends}, libc6
CONTROL

cat > "$WORK/server/etc/tomchat/tomchat-server.env" <<'ENV'
# Configure PostgreSQL before starting the service.
TOMCHAT_DATABASE_URL=
TOMCHAT_LISTEN_ADDR=127.0.0.1:8080
TOMCHAT_REGISTRATION_ENABLED=false
# Set both bootstrap values only for first administrator creation, then remove them.
# TOMCHAT_BOOTSTRAP_ADMIN_USERNAME=
# TOMCHAT_BOOTSTRAP_ADMIN_PASSWORD=
ENV
chmod 0640 "$WORK/server/etc/tomchat/tomchat-server.env"
install -m 0644 "$ROOT/packaging/debian/tomchat-server.service" "$WORK/server/lib/systemd/system/tomchat-server.service"

cat > "$WORK/server/DEBIAN/postinst" <<'POST'
#!/bin/sh
set -e
if ! getent passwd tomchat >/dev/null; then
  useradd --system --home /var/lib/tomchat --create-home --shell /usr/sbin/nologin tomchat
fi
chown root:tomchat /etc/tomchat
chmod 0750 /etc/tomchat
chown root:tomchat /etc/tomchat/tomchat-server.env
chmod 0640 /etc/tomchat/tomchat-server.env
if command -v systemctl >/dev/null 2>&1; then systemctl daemon-reload || true; fi
exit 0
POST
chmod 0755 "$WORK/server/DEBIAN/postinst"

cat > "$WORK/server/DEBIAN/prerm" <<'PRERM'
#!/bin/sh
set -e
if command -v systemctl >/dev/null 2>&1 && { [ "$1" = remove ] || [ "$1" = deconfigure ]; }; then
  systemctl stop tomchat-server.service >/dev/null 2>&1 || true
  systemctl disable tomchat-server.service >/dev/null 2>&1 || true
fi
exit 0
PRERM
chmod 0755 "$WORK/server/DEBIAN/prerm"

dpkg-deb --root-owner-group --build "$WORK/server" "$OUT/tomchat-server_${VERSION}_${ARCH}.deb"
dpkg-deb --root-owner-group --build "$WORK/tui" "$OUT/tomchat-tui_${VERSION}_${ARCH}.deb"
echo "Packages written to $OUT"
