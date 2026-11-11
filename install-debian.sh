#!/usr/bin/env bash
set -Eeuo pipefail

REPOSITORY="${TOMCHAT_REPOSITORY:-https://github.com/Spruhantom/tomos-chat-server.git}"
BRANCH="${TOMCHAT_INSTALL_BRANCH:-migration/tomchat-consolidation}"
TMP_DIR="$(mktemp -d)"
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT
fail() { echo "TomChat installer: $*" >&2; exit 1; }

[[ "$(id -u)" -ne 0 ]] || fail "Run as your normal user; the installer uses sudo when required."
command -v sudo >/dev/null 2>&1 || fail "sudo is required."
command -v curl >/dev/null 2>&1 || fail "curl is required."
[[ -r /etc/os-release ]] || fail "Cannot identify this Linux distribution."
. /etc/os-release
[[ "${ID:-}" == "debian" && "${VERSION_ID:-}" == "13" ]] || fail "This installer currently supports Debian 13 (Trixie)."

echo "==> Installing TomChat build tools"
sudo apt-get update
sudo apt-get install -y build-essential ca-certificates cargo git pkg-config rustc

echo "==> Downloading decentralised TomChat client source"
git clone --depth 1 --branch "$BRANCH" "$REPOSITORY" "$TMP_DIR/tomchat"

echo "==> Building terminal client"
(
  cd "$TMP_DIR/tomchat/tui"
  cargo build --release
)

echo "==> Installing terminal client"
sudo install -Dm0755 "$TMP_DIR/tomchat/tui/target/release/tomchat" /usr/local/bin/tomchat

echo
echo "TomChat terminal client installed."
echo "Run: tomchat"
echo
echo "This is currently a UI foundation. Direct peer discovery, encrypted messaging"
echo "and delivery are still under development; this build does not send messages."
