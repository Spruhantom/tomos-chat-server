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
command -v git >/dev/null 2>&1 || true
[[ -r /etc/os-release ]] || fail "Cannot identify this Linux distribution."
. /etc/os-release
[[ "${ID:-}" == "debian" && "${VERSION_ID:-}" == "13" ]] || fail "This installer currently supports Debian 13 (Trixie)."

echo "==> Installing TomChat build tools"
sudo apt-get update
sudo apt-get install -y build-essential ca-certificates curl git pkg-config

# Keep TomChat's Rust installation separate from ~/.cargo. This avoids
# permission errors caused by an old root-owned Cargo cache and avoids
# relying on Debian's potentially older rustc/cargo packages.
TOMCHAT_DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}/tomchat"
export CARGO_HOME="$TOMCHAT_DATA_HOME/cargo"
export RUSTUP_HOME="$TOMCHAT_DATA_HOME/rustup"
export PATH="$CARGO_HOME/bin:$PATH"

mkdir -p "$CARGO_HOME" "$RUSTUP_HOME"
[[ -w "$CARGO_HOME" && -w "$RUSTUP_HOME" ]] || fail "TomChat Rust directories are not writable: $TOMCHAT_DATA_HOME"

echo "==> Installing the current stable Rust toolchain"
curl --proto '=https' --tlsv1.2 -fsSL https://sh.rustup.rs -o "$TMP_DIR/rustup-init.sh"
sh "$TMP_DIR/rustup-init.sh" -y --profile minimal --default-toolchain stable --no-modify-path
rustc --version
cargo --version

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
