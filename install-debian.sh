#!/usr/bin/env bash
set -Eeuo pipefail

REPOSITORY="https://github.com/Spruhantom/tomos-chat-server.git"
BRANCH="${TOMCHAT_INSTALL_BRANCH:-migration/tomchat-consolidation}"
TMP_DIR="$(mktemp -d)"
KEEP_TEMP="${TOMCHAT_KEEP_TEMP:-false}"
cleanup() {
  if [[ "$KEEP_TEMP" == "true" ]]; then
    echo "Build files kept at: $TMP_DIR"
  else
    rm -rf "$TMP_DIR"
  fi
}
trap cleanup EXIT

fail() { echo "TomChat installer: $*" >&2; exit 1; }

[[ "$(id -u)" -ne 0 ]] || fail "Run this installer as your normal user, not through sudo. It will request sudo when needed."
command -v sudo >/dev/null 2>&1 || fail "sudo is required. Install sudo and ensure your user can administer the system."
command -v curl >/dev/null 2>&1 || fail "curl is required."
command -v git >/dev/null 2>&1 || fail "git is required."
[[ -r /etc/os-release ]] || fail "Cannot identify this Linux distribution."
. /etc/os-release
[[ "${ID:-}" == "debian" && "${VERSION_ID:-}" == "13" ]] || fail "This installer currently supports Debian 13 (Trixie) only."
[[ "$(dpkg --print-architecture)" == "amd64" ]] || fail "This installer currently supports amd64 only."

echo "==> Installing TomChat build dependencies"
sudo apt-get update
sudo apt-get install -y build-essential ca-certificates cargo dpkg-dev git golang-go rustc

echo "==> Downloading TomChat source ($BRANCH)"
git clone --depth 1 --branch "$BRANCH" "$REPOSITORY" "$TMP_DIR/source"

echo "==> Building Debian packages"
(
  cd "$TMP_DIR/source"
  bash packaging/debian/build-packages.sh
)

shopt -s nullglob
packages=("$TMP_DIR"/source/build/deb/*.deb)
(("${#packages[@]}" > 0)) || fail "The package build did not produce any .deb files."

echo "==> Installing TomChat packages"
sudo apt-get install -y "${packages[@]}"

cat <<'DONE'

TomChat packages have been installed.

Server:
  1. Configure PostgreSQL and edit:
       sudoedit /etc/tomchat/tomchat-server.env
  2. Set TOMCHAT_DATABASE_URL before starting the service.
  3. Start it with:
       sudo systemctl enable --now tomchat-server

Terminal client:
  Run: tomchat

Important: the terminal client is currently a UI foundation and is not yet
connected to the server. The server stores opaque message payloads, but a
complete, independently reviewed end-to-end encryption protocol is not yet
implemented. Do not expose the server publicly without TLS and a security
review.
DONE
