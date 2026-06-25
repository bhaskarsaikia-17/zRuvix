#!/usr/bin/env bash
#
# zRuvix management script for Ubuntu/Debian VPS.
#
#   ./setup.sh setup      Install Go + Redis, build, create & start the service
#   ./setup.sh start      Start the service
#   ./setup.sh stop       Stop the service
#   ./setup.sh restart    Restart the service (use after editing .env)
#   ./setup.sh status     Show service status
#   ./setup.sh logs       Follow live logs (Ctrl+C to exit)
#   ./setup.sh env        Edit .env, then restart the service
#   ./setup.sh build      Rebuild the binary and restart the service
#   ./setup.sh uninstall  Stop & remove the systemd service (keeps files)
#
set -euo pipefail

# --- constants ---------------------------------------------------------------
SERVICE_NAME="zruvix"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="${SCRIPT_DIR}/${SERVICE_NAME}"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
ENV_FILE="${SCRIPT_DIR}/.env"
ENV_EXAMPLE="${SCRIPT_DIR}/.env.example"
RUN_USER="${SUDO_USER:-$(whoami)}"
GO_BIN="/usr/local/go/bin/go"

# sudo prefix when not already root
if [ "$(id -u)" -eq 0 ]; then SUDO=""; else SUDO="sudo"; fi

# --- helpers -----------------------------------------------------------------
log()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  ✓\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m  !\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

# Resolve a usable `go` command (PATH or /usr/local/go/bin).
go_cmd() {
  if command -v go >/dev/null 2>&1; then echo "go";
  elif [ -x "$GO_BIN" ]; then echo "$GO_BIN";
  else return 1; fi
}

# --- install steps -----------------------------------------------------------
install_go() {
  if go_cmd >/dev/null 2>&1; then
    ok "Go already installed: $($(go_cmd) version)"
    return
  fi
  log "Installing Go from go.dev ..."
  local arch ostag ver url tmp
  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) die "Unsupported architecture: $(uname -m)" ;;
  esac
  ostag="linux"
  ver="$(curl -fsSL 'https://go.dev/VERSION?m=text' | head -n1)"
  [ -n "$ver" ] || die "Could not determine latest Go version"
  url="https://go.dev/dl/${ver}.${ostag}-${arch}.tar.gz"
  tmp="$(mktemp -d)"
  log "Downloading ${url}"
  curl -fsSL "$url" -o "${tmp}/go.tar.gz" || die "Go download failed"
  $SUDO rm -rf /usr/local/go
  $SUDO tar -C /usr/local -xzf "${tmp}/go.tar.gz"
  rm -rf "$tmp"
  # Make go available on PATH for all future shells.
  echo 'export PATH=$PATH:/usr/local/go/bin' | $SUDO tee /etc/profile.d/go.sh >/dev/null
  export PATH="$PATH:/usr/local/go/bin"
  ok "Installed $(${GO_BIN} version)"
}

install_redis() {
  if command -v redis-server >/dev/null 2>&1; then
    ok "Redis already installed"
  else
    log "Installing Redis ..."
    $SUDO apt-get update -y
    $SUDO apt-get install -y redis-server
  fi
  # Redis binds to 127.0.0.1:6379 by default — exactly what the API uses.
  $SUDO systemctl enable redis-server >/dev/null 2>&1 || true
  $SUDO systemctl restart redis-server || $SUDO systemctl restart redis || true
  ok "Redis running on 127.0.0.1:6379"
}

ensure_env() {
  if [ ! -f "$ENV_FILE" ]; then
    if [ -f "$ENV_EXAMPLE" ]; then
      cp "$ENV_EXAMPLE" "$ENV_FILE"
      warn "Created .env from .env.example — set your BOT_TOKEN with: ./setup.sh env"
    else
      die ".env.example not found next to setup.sh"
    fi
  else
    ok ".env already exists (left untouched)"
  fi
}

build_binary() {
  local go
  go="$(go_cmd)" || die "Go is not installed. Run: ./setup.sh setup"
  log "Downloading Go dependencies ..."
  ( cd "$SCRIPT_DIR" && "$go" mod download )
  log "Building ${SERVICE_NAME} binary ..."
  ( cd "$SCRIPT_DIR" && "$go" build -o "$BINARY" ./cmd/zruvix )
  $SUDO chown "$RUN_USER":"$RUN_USER" "$BINARY" 2>/dev/null || true
  ok "Built $BINARY"
}

write_service() {
  log "Creating systemd service ($SERVICE_FILE) ..."
  $SUDO tee "$SERVICE_FILE" >/dev/null <<EOF
[Unit]
Description=zRuvix - Discord presence API & WebSocket gateway
After=network.target redis-server.service
Wants=redis-server.service

[Service]
Type=simple
User=${RUN_USER}
WorkingDirectory=${SCRIPT_DIR}
# .env is read from WorkingDirectory by the app; also exported here so plain
# systemd env is populated too. The leading - makes it optional.
EnvironmentFile=-${ENV_FILE}
ExecStart=${BINARY}
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
  $SUDO systemctl daemon-reload
  $SUDO systemctl enable "$SERVICE_NAME" >/dev/null 2>&1 || true
  ok "Service installed and enabled (starts on boot)"
}

# --- subcommands -------------------------------------------------------------
cmd_setup() {
  command -v curl >/dev/null 2>&1 || { $SUDO apt-get update -y && $SUDO apt-get install -y curl; }
  install_go
  install_redis
  ensure_env
  build_binary
  write_service
  $SUDO systemctl restart "$SERVICE_NAME"
  echo
  ok "Setup complete. zRuvix is running 24/7 on port ${PORT:-4001}."
  warn "If you haven't yet, set your bot token:  ./setup.sh env"
  echo  "Check it:  curl http://localhost:4001/"
}

cmd_start()   { $SUDO systemctl start   "$SERVICE_NAME"; ok "started"; }
cmd_stop()    { $SUDO systemctl stop    "$SERVICE_NAME"; ok "stopped"; }
cmd_restart() { $SUDO systemctl restart "$SERVICE_NAME"; ok "restarted"; }
cmd_status()  { $SUDO systemctl status  "$SERVICE_NAME" --no-pager; }
cmd_logs()    { $SUDO journalctl -u "$SERVICE_NAME" -f --no-pager; }

cmd_env() {
  [ -f "$ENV_FILE" ] || ensure_env
  "${EDITOR:-nano}" "$ENV_FILE"
  log "Restarting to apply .env changes ..."
  $SUDO systemctl restart "$SERVICE_NAME"
  ok "Applied. Tail logs with: ./setup.sh logs"
}

cmd_build() {
  build_binary
  $SUDO systemctl restart "$SERVICE_NAME" 2>/dev/null || warn "Service not installed yet; run ./setup.sh setup"
  ok "Rebuilt and restarted"
}

cmd_uninstall() {
  $SUDO systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  $SUDO systemctl disable "$SERVICE_NAME" 2>/dev/null || true
  $SUDO rm -f "$SERVICE_FILE"
  $SUDO systemctl daemon-reload
  ok "Service removed (binary, .env and source left in place)"
}

usage() {
  sed -n '3,20p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

# --- dispatch ----------------------------------------------------------------
case "${1:-}" in
  setup)     cmd_setup ;;
  start)     cmd_start ;;
  stop)      cmd_stop ;;
  restart)   cmd_restart ;;
  status)    cmd_status ;;
  logs)      cmd_logs ;;
  env)       cmd_env ;;
  build)     cmd_build ;;
  uninstall) cmd_uninstall ;;
  ""|-h|--help|help) usage ;;
  *) die "Unknown command: $1 (run ./setup.sh help)" ;;
esac
