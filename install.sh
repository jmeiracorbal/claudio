#!/bin/sh
# claudio installer
# Usage: curl -sSfL https://raw.githubusercontent.com/jmeiracorbal/claudio/main/install.sh | sh

set -e

REPO="jmeiracorbal/claudio"
BINARY="${REPO##*/}"
INSTALL_DIR="/usr/local/bin"

# ── helpers ────────────────────────────────────────────────────────────────────

info() { printf "\033[1;34m[claudio]\033[0m %s\n" "$*"; }
ok()   { printf "\033[1;32m[claudio]\033[0m %s\n" "$*"; }
err()  { printf "\033[1;31m[claudio]\033[0m %s\n" "$*" >&2; exit 1; }

# ── platform ───────────────────────────────────────────────────────────────────

detect_platform() {
  local os arch

  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux)  os="linux"  ;;
    *) err "Unsupported OS: $(uname -s)" ;;
  esac

  case "$(uname -m)" in
    x86_64)        arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) err "Unsupported architecture: $(uname -m)" ;;
  esac

  echo "${os}-${arch}"
}

# ── fetch ──────────────────────────────────────────────────────────────────────

fetch() {
  local url="$1" dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -sSfL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
  else
    err "curl or wget required"
  fi
}

fetch_stdout() {
  if command -v curl >/dev/null 2>&1; then
    curl -sSfL "$1"
  else
    wget -qO- "$1"
  fi
}

# ── version ────────────────────────────────────────────────────────────────────

fetch_latest_version() {
  local version
  version=$(fetch_stdout "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  [ -z "$version" ] && err "Could not resolve latest release version"
  echo "$version"
}

# ── download ───────────────────────────────────────────────────────────────────

download_binary() {
  local version="$1" platform="$2"
  local base_url="https://github.com/${REPO}/releases/download/${version}"
  local binary_url="${base_url}/${BINARY}-${platform}"
  local checksum_url="${base_url}/${BINARY}-${platform}.sha256"

  info "Downloading ${BINARY} ${version} for ${platform}..."

  mkdir -p "$INSTALL_DIR"

  local tmp
  tmp=$(mktemp)
  trap 'rm -f "$tmp"' EXIT

  fetch "$binary_url" "$tmp" || err "Download failed: ${binary_url}"

  local expected actual
  expected=$(fetch_stdout "$checksum_url" | awk '{print $1}')
  if command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$tmp" | awk '{print $1}')
  elif command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$tmp" | awk '{print $1}')
  else
    err "shasum or sha256sum required to verify the download"
  fi

  [ "$expected" = "$actual" ] || err "Checksum mismatch — aborting"
  chmod +x "$tmp"

  if [ -w "$INSTALL_DIR" ]; then
    mv "$tmp" "${INSTALL_DIR}/${BINARY}"
  else
    sudo mv "$tmp" "${INSTALL_DIR}/${BINARY}"
  fi

  ok "Installed: ${INSTALL_DIR}/${BINARY}"
}

# ── PATH check ─────────────────────────────────────────────────────────────────

check_path() {
  if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
    printf "\033[1;33m[claudio]\033[0m %s is not in your PATH.\n" "$INSTALL_DIR"
    printf "\033[1;33m[claudio]\033[0m Add to your shell profile:\n"
    printf "\033[1;33m[claudio]\033[0m   export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR"
  fi
}

# ── main ───────────────────────────────────────────────────────────────────────

main() {
  local platform version

  platform=$(detect_platform)
  version=$(fetch_latest_version)

  info "Latest release: ${version}"

  download_binary "$version" "$platform"
  check_path

  ok "Done. Run 'claudio --help' to get started."
}

main "$@"
