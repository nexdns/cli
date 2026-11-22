#!/bin/sh
# NexDNS CLI installer
# Usage: curl -sL https://get.nexdns.tech/cli | sh
set -e

# The archive and its checksums come from the host that served this script.
HOST="get.nexdns.tech"
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

echo "Detecting latest version..."
LATEST=$(curl -fsSL "https://${HOST}/cli/latest" | tr -d ' \t\r\n')

if [ -z "$LATEST" ]; then
    echo "Error: could not detect latest version. Check https://${HOST}/cli/latest"
    exit 1
fi

# Release archives are named nexdns_<version>_<os>_<arch>.tar.gz (version has no leading 'v')
VERSION="${LATEST#v}"
FILENAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://${HOST}/cli/download/${LATEST}"
URL="${BASE_URL}/${FILENAME}"

echo "Downloading ${BINARY} ${LATEST} for ${OS}/${ARCH}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

if ! curl -fsSL "$URL" -o "${TMP_DIR}/${FILENAME}"; then
    echo "Error: could not download ${URL}"
    exit 1
fi

# Nothing is installed unless the checksum can be verified.
echo "Verifying checksum..."
if ! curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMP_DIR}/checksums.txt"; then
    echo "Error: could not download ${BASE_URL}/checksums.txt"
    exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
    SUM=$(sha256sum "${TMP_DIR}/${FILENAME}" | cut -d" " -f1)
elif command -v shasum >/dev/null 2>&1; then
    SUM=$(shasum -a 256 "${TMP_DIR}/${FILENAME}" | cut -d" " -f1)
else
    echo "Error: no sha256 tool available, so ${FILENAME} cannot be verified."
    echo "  Install coreutils (sha256sum) or perl-shasum and run this again."
    exit 1
fi

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
