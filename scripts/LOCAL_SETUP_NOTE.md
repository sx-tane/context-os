# Local Setup Note (Ubuntu)

Use these commands to install the required tooling for this repository.

## 1) Install base utilities

```bash
sudo apt update
sudo apt install -y curl wget tar xz-utils git build-essential ca-certificates
```

## 2) Install Go 1.24.13

```bash
cd /tmp
wget https://go.dev/dl/go1.24.13.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.13.linux-amd64.tar.gz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
export PATH=/usr/local/go/bin:$PATH
```

## 3) Install Bun

```bash
curl -fsSL https://bun.sh/install | bash
echo 'export BUN_INSTALL="$HOME/.bun"' >> ~/.bashrc
echo 'export PATH="$BUN_INSTALL/bin:$PATH"' >> ~/.bashrc
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"
```

## 4) Install Python 3.12 and tools

```bash
sudo apt install -y python3.12 python3.12-venv python3-pip
```

## 5) Install uv

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
export PATH="$HOME/.local/bin:$PATH"
```

## 6) Verify installations

```bash
go version
bun --version
python3.12 --version
uv --version
```

## 7) Validate this repo

```bash
cd /workspaces/context-os
go mod tidy
go test ./...
cd apps/frontend && bun install && bun run check
cd /workspaces/context-os/apps/ai-worker && uv sync
```
