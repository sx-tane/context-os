#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_VERSION="1.24.13"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"

info() {
  printf '[setup] %s\n' "$1"
}

require_linux() {
  if [[ "$(uname -s)" != "Linux" ]]; then
    echo "This setup script currently supports Linux only." >&2
    exit 1
  fi
}

require_sudo() {
  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo is required for system package installation." >&2
    exit 1
  fi
}

append_if_missing() {
  local line="$1"
  local file="$2"

  touch "$file"
  if ! grep -Fqx "$line" "$file"; then
    printf '%s\n' "$line" >> "$file"
  fi
}

install_base_packages() {
  info "Installing base utilities"
  sudo apt update
  sudo apt install -y curl wget tar xz-utils git build-essential ca-certificates
}

install_go() {
  info "Installing Go ${GO_VERSION}"
  (
    cd /tmp
    wget -q "$GO_URL"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "$GO_TARBALL"
    rm -f "$GO_TARBALL"
  )

  export PATH="/usr/local/go/bin:$PATH"
  append_if_missing 'export PATH=/usr/local/go/bin:$PATH' "$HOME/.bashrc"
}

install_bun() {
  info "Installing Bun"
  curl -fsSL https://bun.sh/install | bash

  append_if_missing 'export BUN_INSTALL="$HOME/.bun"' "$HOME/.bashrc"
  append_if_missing 'export PATH="$BUN_INSTALL/bin:$PATH"' "$HOME/.bashrc"

  export BUN_INSTALL="$HOME/.bun"
  export PATH="$BUN_INSTALL/bin:$PATH"
}

install_python() {
  info "Installing Python 3.12 and tooling"
  sudo apt install -y python3.12 python3.12-venv python3-pip
}

install_uv() {
  info "Installing uv"
  curl -LsSf https://astral.sh/uv/install.sh | sh

  append_if_missing 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.bashrc"
  export PATH="$HOME/.local/bin:$PATH"
}

verify_tools() {
  info "Verifying installed tool versions"
  go version
  bun --version
  python3.12 --version
  uv --version
}

validate_repo() {
  info "Running repository validation checks"
  (
    cd "$ROOT_DIR"
    go mod tidy
    go test ./...
  )

  (
    cd "$ROOT_DIR/apps/frontend"
    bun install
    bun run check
  )

  (
    cd "$ROOT_DIR/apps/ai-worker"
    uv sync
  )
}

main() {
  require_linux
  require_sudo

  install_base_packages
  install_go
  install_bun
  install_python
  install_uv
  verify_tools
  validate_repo

  info "Setup complete. Restart your shell to reload PATH from ~/.bashrc."
}

main "$@"