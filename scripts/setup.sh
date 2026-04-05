#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

info()  { echo -e "${BOLD}$1${NC}"; }
ok()    { echo -e "  ${GREEN}✓${NC} $1"; }
warn()  { echo -e "  ${YELLOW}!${NC} $1"; }
fail()  { echo -e "  ${RED}✗${NC} $1"; }

info "Setting up dmud development environment..."
echo

# --- Go ---
if ! command -v go &>/dev/null; then
    fail "Go is not installed."
    echo "    Install Go 1.22+: https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | sed 's/go//')
GO_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)

if [ "$GO_MAJOR" -lt 1 ] || { [ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 22 ]; }; then
    fail "Go $GO_VERSION found, but 1.22+ is required."
    echo "    Upgrade: https://go.dev/dl/"
    exit 1
fi
ok "Go $GO_VERSION"

# --- Dependencies ---
info "Downloading Go modules..."
go mod download
ok "Dependencies downloaded"

# --- Air (live reload) ---
AIR_PATH="$(go env GOPATH)/bin/air"
if [ -x "$AIR_PATH" ]; then
    ok "air already installed"
else
    info "Installing air (live reload)..."
    go install github.com/air-verse/air@latest
    ok "air installed"
fi

# --- Directories ---
mkdir -p bin tmp
ok "Created bin/ and tmp/ directories"

# --- Docker (optional) ---
echo
if command -v docker &>/dev/null; then
    ok "Docker found (optional, used for make dc-up)"
else
    warn "Docker not found (optional, only needed for make dc-up)"
fi

# --- Summary ---
echo
echo -e "${GREEN}${BOLD}Setup complete!${NC}"
echo
echo "  Get started:"
echo "    make dev        Start dev server with hot reload"
echo "    make test       Run tests"
echo "    make build      Build binary"
echo
echo "  Environment variables (see .env.example):"
echo "    DMUD_LOG_LEVEL=debug    Verbose logging"
echo "    DMUD_PERSISTENCE=redis  Enable Redis persistence"
echo
