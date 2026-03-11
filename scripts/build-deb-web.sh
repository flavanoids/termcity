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

# Systemd service
mkdir -p "$PKG/lib/systemd/system"
cp scripts/termcity-web.service "$PKG/lib/systemd/system/termcity-web.service"

# Default config (won't overwrite existing on upgrade)
mkdir -p "$PKG/etc/default"
cp scripts/termcity-web.default "$PKG/etc/default/termcity-web"

# Debian control
mkdir -p "$PKG/DEBIAN"
cat > "$PKG/DEBIAN/control" << EOF
Package: termcity-web
Version: $VERSION
Section: web
Priority: optional
Architecture: $ARCH
Maintainer: TermCity <termcity@localhost>
Description: TermCity web — 911 incident map in your browser
 Browser-based 911 incident viewer with history tracking.
 Enter a US zip code to see fire, EMS, and police incidents
 on an OpenStreetMap map. Stores incident history in a local
 SQLite database for 7-day lookback.
 .
 Configure zip code in /etc/default/termcity-web, then:
   systemctl enable --now termcity-web
EOF

# Mark /etc/default/termcity-web as a conffile (preserved on upgrade)
cat > "$PKG/DEBIAN/conffiles" << EOF
/etc/default/termcity-web
EOF

# postinst: enable and start the service
cat > "$PKG/DEBIAN/postinst" << 'EOF'
#!/bin/sh
set -e
if [ "$1" = "configure" ]; then
    systemctl daemon-reload
    systemctl enable termcity-web
    systemctl restart termcity-web || true
fi
EOF
chmod 755 "$PKG/DEBIAN/postinst"

# prerm: stop and disable the service before removal
cat > "$PKG/DEBIAN/prerm" << 'EOF'
#!/bin/sh
set -e
if [ "$1" = "remove" ]; then
    systemctl stop termcity-web || true
    systemctl disable termcity-web || true
fi
EOF
chmod 755 "$PKG/DEBIAN/prerm"

dpkg-deb --build "$PKG"
rm -rf "$PKG"
echo "Built: ${PKG}.deb"
