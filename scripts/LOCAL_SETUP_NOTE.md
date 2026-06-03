# Local Setup Note (Ubuntu)

Use these commands to install the required tooling for this repository.
Alternatively, run `./scripts/setup-local.sh` which performs all steps below automatically.

## 1) Install base utilities

```bash
sudo apt update
sudo apt install -y curl wget tar xz-utils unzip git build-essential ca-certificates nodejs
```

## 2) Install Go 1.24.13

```bash
cd /tmp
wget https://go.dev/dl/go1.24.13.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.13.linux-amd64.tar.gz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
export PATH=/usr/local/go/bin:$PATH

# Install goimports formatter
go install golang.org/x/tools/cmd/goimports@latest
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

## 3) Install Bun

```bash
curl -fsSL https://bun.sh/install | bash
echo 'export BUN_INSTALL="$HOME/.bun"' >> ~/.bashrc
echo 'export PATH="$BUN_INSTALL/bin:$PATH"' >> ~/.bashrc
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"
```

## 4) Install Python tooling

```bash
sudo apt install -y python3-venv python3-pip
```

## 5) Install uv

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
export PATH="$HOME/.local/bin:$PATH"
```

## 6) Install Codex CLI

```bash
sudo npm install -g @openai/codex

# Install GitHub, Atlassian Rovo, and Slack plugins
codex plugin add github@openai-curated
codex plugin add atlassian-rovo@openai-curated
codex plugin add slack@openai-curated

# Log in (opens browser; use --device-auth in headless/SSH environments)
codex login
```

## 7) Verify tools and validate the repository

```bash
go version
bun --version
uv --version
codex --version

# Verifies the worker can run under uv-managed Python 3.12
cd apps/ai-worker && uv run python --version

# From the repo root
go mod tidy
go test ./...

cd apps/frontend && bun install && bun run check
cd apps/ai-worker && uv sync
```

## After setup

Restart your shell or run `source ~/.bashrc` to reload the updated PATH, then start all services:

```bash
./scripts/start-all.sh
```
