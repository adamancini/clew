#!/bin/bash
# SessionStart hook for clew plugin
# Symlinks the appropriate clew binary to ~/.local/bin/clew based on platform/arch
#
# This script is idempotent - safe to run multiple times.
# It will:
# 1. Detect the current platform (darwin/linux) and architecture (arm64/amd64)
# 2. Create ~/.local/bin if it doesn't exist
# 3. Symlink the appropriate binary from bin/ to ~/.local/bin/clew
# 4. Make the binary executable
# 5. Warn if ~/.local/bin is not in PATH

set -euo pipefail

# Get the plugin root directory
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(dirname "$(dirname "$0")")}"
BIN_DIR="${PLUGIN_ROOT}/bin"
INSTALL_DIR="${HOME}/.local/bin"
BINARY_NAME="clew"

# Detect platform
detect_platform() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        darwin) echo "darwin" ;;
        linux)  echo "linux" ;;
        *)
            echo "Error: Unsupported platform: $os" >&2
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64)  echo "amd64" ;;
        amd64)   echo "amd64" ;;
        arm64)   echo "arm64" ;;
        aarch64) echo "arm64" ;;
        *)
            echo "Error: Unsupported architecture: $arch" >&2
            exit 1
            ;;
    esac
}

# Check if PATH includes install directory
check_path() {
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        echo "Warning: ${INSTALL_DIR} is not in your PATH" >&2
        echo "Add this to your shell profile:" >&2
        echo "  export PATH=\"\${HOME}/.local/bin:\${PATH}\"" >&2
    fi
}

# Main installation logic
main() {
    local platform arch binary_path

    platform="$(detect_platform)"
    arch="$(detect_arch)"
    binary_path="${BIN_DIR}/${BINARY_NAME}-${platform}-${arch}"

    # Check if source binary exists
    if [[ ! -f "$binary_path" ]]; then
        echo "Error: Binary not found: $binary_path" >&2
        echo "Available binaries in ${BIN_DIR}:" >&2
        ls -la "$BIN_DIR" 2>/dev/null || echo "  (bin directory does not exist)" >&2
        exit 1
    fi

    # Create install directory if needed
    if [[ ! -d "$INSTALL_DIR" ]]; then
        echo "Creating directory: ${INSTALL_DIR}"
        mkdir -p "$INSTALL_DIR"
    fi

    local target="${INSTALL_DIR}/${BINARY_NAME}"

    # Check if already correctly symlinked
    if [[ -L "$target" ]]; then
        local current_target
        current_target="$(readlink "$target")"
        if [[ "$current_target" == "$binary_path" ]]; then
            echo "clew is already correctly installed at ${target}"
            check_path
            exit 0
        fi
        # Remove old symlink
        echo "Updating existing symlink..."
        rm -f "$target"
    elif [[ -e "$target" ]]; then
        # File exists but is not a symlink - back it up
        echo "Backing up existing ${target} to ${target}.bak"
        mv "$target" "${target}.bak"
    fi

    # Create symlink
    echo "Installing clew for ${platform}/${arch}..."
    ln -s "$binary_path" "$target"

    # Ensure executable
    chmod +x "$binary_path"

    echo "clew installed successfully at ${target}"
    echo "  Platform: ${platform}"
    echo "  Architecture: ${arch}"
    echo "  Binary: ${binary_path}"

    # Verify installation
    if command -v clew >/dev/null 2>&1; then
        echo "  Version: $(clew --version 2>/dev/null || echo 'unknown')"
    else
        check_path
    fi
}

main "$@"
