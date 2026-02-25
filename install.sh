#!/bin/sh
# install.sh — install agent-sync on macOS or Linux
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/bianoble/agent-sync/main/install.sh | sh
#
# Environment variables:
#   VERSION      — version to install (default: latest)
#   INSTALL_DIR  — directory to install to (default: /usr/local/bin or ~/.local/bin)

set -eu

REPO="bianoble/agent-sync"
BINARY="agent-sync"

main() {
    need_cmd curl
    need_cmd uname

    os="$(detect_os)"
    arch="$(detect_arch)"

    if [ -z "${VERSION:-}" ]; then
        VERSION="$(get_latest_version)"
    fi

    # Strip leading v for display.
    version_display="${VERSION#v}"
    printf "Installing %s v%s (%s/%s)\n" "$BINARY" "$version_display" "$os" "$arch"

    install_dir="${INSTALL_DIR:-}"
    if [ -z "$install_dir" ]; then
        if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
            install_dir="/usr/local/bin"
        else
            install_dir="${HOME}/.local/bin"
            mkdir -p "$install_dir"
        fi
    fi

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    archive_name="${BINARY}_${os}_${arch}.tar.gz"
    checksums_name="checksums.txt"

    base_url="https://github.com/${REPO}/releases/download/${VERSION}"

    printf "Downloading %s...\n" "$archive_name"
    download "${base_url}/${archive_name}" "${tmpdir}/${archive_name}"
    download "${base_url}/${checksums_name}" "${tmpdir}/${checksums_name}"

    printf "Verifying checksum...\n"
    verify_checksum "$tmpdir" "$archive_name"

    printf "Extracting...\n"
    tar -xzf "${tmpdir}/${archive_name}" -C "$tmpdir"

    install -m 755 "${tmpdir}/${BINARY}" "${install_dir}/${BINARY}"
    printf "Installed %s to %s/%s\n" "$BINARY" "$install_dir" "$BINARY"

    # Verify the binary works.
    if command -v "$BINARY" >/dev/null 2>&1; then
        printf "\n"
        "$BINARY" version
    elif [ -x "${install_dir}/${BINARY}" ]; then
        printf "\n"
        "${install_dir}/${BINARY}" version
        printf "\nNote: %s is not in your PATH. Add it with:\n" "$install_dir"
        printf "  export PATH=\"%s:\$PATH\"\n" "$install_dir"
    fi
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       err "Unsupported OS: $(uname -s). Use install.ps1 for Windows." ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             err "Unsupported architecture: $(uname -m)" ;;
    esac
}

get_latest_version() {
    url="https://api.github.com/repos/${REPO}/releases/latest"
    version="$(curl -sSL "$url" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
    if [ -z "$version" ]; then
        err "Could not determine latest version from GitHub API"
    fi
    echo "$version"
}

download() {
    url="$1"
    dest="$2"
    if ! curl -fsSL -o "$dest" "$url"; then
        err "Failed to download: $url"
    fi
}

verify_checksum() {
    dir="$1"
    file="$2"
    expected="$(grep "$file" "${dir}/checksums.txt" | awk '{print $1}')"
    if [ -z "$expected" ]; then
        err "Checksum not found for $file in checksums.txt"
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual="$(sha256sum "${dir}/${file}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        actual="$(shasum -a 256 "${dir}/${file}" | awk '{print $1}')"
    else
        printf "Warning: no sha256sum or shasum found, skipping checksum verification\n"
        return
    fi

    if [ "$actual" != "$expected" ]; then
        err "Checksum mismatch for $file (expected $expected, got $actual)"
    fi
}

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        err "Required command not found: $1"
    fi
}

err() {
    printf "Error: %s\n" "$1" >&2
    exit 1
}

main
