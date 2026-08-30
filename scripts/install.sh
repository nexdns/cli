#!/bin/sh
# NexDNS CLI installer
# Usage: curl -sL https://get.nexdns.tech/cli | sh
set -e

# Everything is fetched from the instance that served this script. The script
# itself already runs unconditionally on the caller's machine, so the binary it
# installs should not come from a different party than the script did - and an
# install must not fail because a third-party host is unreachable from the
# caller's network.
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

# Ask the instance which version it holds. It answers with what is on its own
# disk, so the version named here is one it can also serve; an instance that is
# behind installs an older CLI rather than pointing at an archive it lacks.
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
    echo "  ${HOST} named ${LATEST} moments ago. Either it is being"
    echo "  re-published right now - retry in a minute - or that release was"
    echo "  built without a ${OS}/${ARCH} archive."
    exit 1
fi

# Verify against the checksums file published beside the archive. Every branch
# below ends the run rather than continuing: this script is piped to a shell, so
# bytes that could not be checked do not get installed. sha256sum is the GNU
# spelling, shasum the BSD/macOS one.
echo "Verifying checksum..."
if ! curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMP_DIR}/checksums.txt"; then
    echo "Error: could not download ${BASE_URL}/checksums.txt"
    echo "  ${HOST} named ${LATEST} moments ago, and it only names a version"
    echo "  whose manifest is in place, so this is most likely a release being"
    echo "  re-published right now. Retry in a minute. If it persists, the"
    echo "  server's artifacts directory is incomplete and needs re-syncing."
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
    echo "  The manifest on ${HOST} does not describe the archive beside it."
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
