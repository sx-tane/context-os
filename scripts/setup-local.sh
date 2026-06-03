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
  sudo apt install -y curl wget tar xz-utils git build-essential ca-certificates nodejs
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

  info "Installing goimports (go formatter used by VS Code)"
  go install golang.org/x/tools/cmd/goimports@latest
  append_if_missing 'export PATH="$HOME/go/bin:$PATH"' "$HOME/.bashrc"
  export PATH="$HOME/go/bin:$PATH"
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
  info "Installing Python tooling"
  sudo apt install -y python3-venv python3-pip
}

install_uv() {
  info "Installing uv"
  curl -LsSf https://astral.sh/uv/install.sh | sh

  append_if_missing 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.bashrc"
  export PATH="$HOME/.local/bin:$PATH"
}

# Returns 0 (true) when running inside a headless, remote, or SSH environment.
is_headless() {
  [[ -n "${CODESPACES:-}" ]]                        && return 0
  [[ -n "${VSCODE_REMOTE_CONTAINERS_SESSION:-}" ]]  && return 0
  [[ -n "${SSH_TTY:-}" ]]                            && return 0
  [[ -n "${SSH_CONNECTION:-}" ]]                     && return 0
  [[ -z "${DISPLAY:-}" ]]                            && return 0
  return 1
}

install_codex() {
  info "Installing Codex CLI"
  if ! command -v npm >/dev/null 2>&1; then
    echo "npm is required to install Codex CLI. Install nodejs (which provides npm on NodeSource) or add npm to PATH." >&2
    exit 1
  fi

  if npm root -g >/dev/null 2>&1 && [[ -w "$(npm root -g)" ]]; then
    npm install -g @openai/codex
  else
    sudo npm install -g @openai/codex
  fi

  info "Installing Codex GitHub, Atlassian Rovo, Slack, and Google Drive plugins"
  codex plugin add github@openai-curated >/dev/null 2>&1 || \
    echo "[warn] Could not install GitHub Codex plugin." >&2
  codex plugin add atlassian-rovo@openai-curated >/dev/null 2>&1 || \
    echo "[warn] Could not install Atlassian Rovo Codex plugin." >&2
  codex plugin add slack@openai-curated >/dev/null 2>&1 || \
    echo "[warn] Could not install Slack Codex plugin." >&2
  codex plugin add google-drive@openai-curated >/dev/null 2>&1 || \
    echo "[warn] Could not install Google Drive Codex plugin." >&2

  if ! codex login status >/dev/null 2>&1; then
    if is_headless; then
      info "Headless/remote environment detected — starting Codex device auth login..."
      echo "     Visit the URL shown below to approve access in your browser."
      codex login --device-auth
    else
      info "Starting Codex login — your browser will open the authorization page..."
      codex login
    fi
  else
    info "Codex CLI already logged in."
  fi
}

verify_tools() {
  info "Verifying installed tool versions"
  go version
  bun --version
  uv --version
  codex --version
  (
    cd "$ROOT_DIR/apps/ai-worker"
    uv run python --version
  )
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
  install_codex
  verify_tools
  validate_repo

  info "Setup complete. Restart your shell to reload PATH from ~/.bashrc."
}

main "$@"
