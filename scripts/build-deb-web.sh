#!/bin/sh
# Build termcity-web binary and create a .deb package.
# Run from the repository root.

set -e
VERSION="${TERMCITY_WEB_VERSION:-1.0.0}"
ARCH="${DEB_ARCH:-amd64}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Build binary for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o termcity-web.linux-amd64 ./cmd/termcity-web

# Package layout
PKG="termcity-web_${VERSION}_${ARCH}"
rm -rf "$PKG"
mkdir -p "$PKG/usr/bin"
mv termcity-web.linux-amd64 "$PKG/usr/bin/termcity-web"
chmod 755 "$PKG/usr/bin/termcity-web"

# Debian control (override Architecture if cross-building)
mkdir -p "$PKG/DEBIAN"
cat > "$PKG/DEBIAN/control" << EOF
Package: termcity-web
Version: $VERSION
Section: web
Priority: optional
Architecture: $ARCH
Maintainer: TermCity <termcity@localhost>
Description: TermCity web — 911 incident map in your browser
 Browser-based 911 incident viewer. Enter a US zip code to see
 fire, EMS, and police incidents on an OpenStreetMap map.
 Runs in the background and prints the URL and port on startup.
 .
 Use -foreground to run in foreground; -port N to set port.
EOF

dpkg-deb --build "$PKG"
rm -rf "$PKG"
echo "Built: ${PKG}.deb"
