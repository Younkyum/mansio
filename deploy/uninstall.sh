#!/usr/bin/env bash
set -euo pipefail

OS="$(uname -s)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

echo ""
echo "  Uninstalling Mansio..."
echo ""

case "$OS" in
    Linux)
        if [[ $EUID -ne 0 ]]; then
            error "Run as root: sudo bash uninstall.sh"
        fi

        USER="${SUDO_USER:-$(whoami)}"

        if systemctl is-active --quiet "mansio@${USER}" 2>/dev/null; then
            info "Stopping service..."
            systemctl stop "mansio@${USER}"
        fi

        if systemctl is-enabled --quiet "mansio@${USER}" 2>/dev/null; then
            info "Disabling service..."
            systemctl disable "mansio@${USER}"
        fi

        if [[ -f /etc/systemd/system/mansio@.service ]]; then
            info "Removing systemd unit..."
            rm -f /etc/systemd/system/mansio@.service
            systemctl daemon-reload
        fi

        if [[ -f /usr/local/bin/mansio ]]; then
            info "Removing binary..."
            rm -f /usr/local/bin/mansio
        fi

        warn "Data directory /var/lib/mansio was NOT removed."
        warn "To delete all data: sudo rm -rf /var/lib/mansio"
        ;;

    Darwin)
        PLIST="${HOME}/Library/LaunchAgents/com.mansio.mansio.plist"

        if [[ -f "$PLIST" ]]; then
            info "Unloading launchd service..."
            launchctl unload "$PLIST" 2>/dev/null || true
            rm -f "$PLIST"
            info "Removed launchd plist"
        fi

        if [[ -f /usr/local/bin/mansio ]]; then
            info "Removing binary (requires sudo)..."
            sudo rm -f /usr/local/bin/mansio
        fi

        warn "Data directory ~/.local/share/mansio was NOT removed."
        warn "To delete all data: rm -rf ~/.local/share/mansio"
        warn "To delete logs: rm -rf ~/Library/Logs/mansio"
        ;;

    *)
        warn "Unknown OS: ${OS}"
        warn "Manually remove /usr/local/bin/mansio"
        ;;
esac

echo ""
info "Uninstall complete."
