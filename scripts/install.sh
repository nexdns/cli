#!/bin/sh
# NexDNS CLI installer
# Usage: curl -sL https://get.nexdns.tech/cli | sh
set -e

REPO="nexdns/cli"
BINARY="nexdns"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    i386|i686)     ARCH="386" ;;
    *)
        echo "Error: unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    linux|darwin) ;;
    mingw*|msys*|cygwin*)
        echo "Error: use the Windows installer or download manually from releases."
        exit 1
        ;;
    *)
        echo "Error: unsupported OS: $OS"
        exit 1
        ;;
esac

# Get latest release tag from the GitHub API
echo "Detecting latest version..."
LATEST=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep -o '"tag_name":[[:space:]]*"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$LATEST" ]; then
    echo "Error: could not detect latest version. Check https://github.com/${REPO}/releases"
    exit 1
fi

# Release archives are named nexdns_<version>_<os>_<arch>.tar.gz (version has no leading 'v')
VERSION="${LATEST#v}"
FILENAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${FILENAME}"

echo "Downloading ${BINARY} ${LATEST} for ${OS}/${ARCH}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$URL" -o "${TMP_DIR}/${FILENAME}"

# Verify against the checksums file published with the release. sha256sum is
# GNU, shasum is the BSD/macOS spelling; if neither exists, say so rather than
# pretending the download was checked.
echo "Verifying checksum..."
if curl -fsSL "https://github.com/${REPO}/releases/download/${LATEST}/checksums.txt" \
        -o "${TMP_DIR}/checksums.txt"; then
    if command -v sha256sum >/dev/null 2>&1; then
        SUM=$(sha256sum "${TMP_DIR}/${FILENAME}" | cut -d" " -f1)
    elif command -v shasum >/dev/null 2>&1; then
        SUM=$(shasum -a 256 "${TMP_DIR}/${FILENAME}" | cut -d" " -f1)
    else
        SUM=""
        echo "Warning: no sha256 tool found, skipping verification"
    fi

    if [ -n "$SUM" ]; then
        EXPECTED=$(grep " ${FILENAME}\$" "${TMP_DIR}/checksums.txt" | cut -d" " -f1)
        if [ -z "$EXPECTED" ]; then
            echo "Error: ${FILENAME} is not listed in checksums.txt"
            exit 1
        fi
        if [ "$SUM" != "$EXPECTED" ]; then
            echo "Error: checksum mismatch for ${FILENAME}"
            echo "  expected: ${EXPECTED}"
            echo "  actual:   ${SUM}"
            exit 1
        fi
    fi
else
    echo "Warning: checksums.txt could not be downloaded, skipping verification"
fi

echo "Extracting..."
tar -xzf "${TMP_DIR}/${FILENAME}" -C "$TMP_DIR"

echo "Installing to ${INSTALL_DIR}/${BINARY}..."
chmod +x "${TMP_DIR}/${BINARY}"
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
echo "${BINARY} ${LATEST} installed successfully!"
echo ""
echo "Get started:"
echo "  ${BINARY} auth token <YOUR_API_TOKEN>"
echo "  ${BINARY} zone list"
echo ""
echo "Create an API key at https://nexdns.tech/settings/api-keys"
