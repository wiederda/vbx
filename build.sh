#!/bin/bash
set -e

PRODUCT_NAME="vbx"
VERSION_FILE="build_number.txt"

# -----------------------
# Clean
# -----------------------
echo "Cleaning old build folders..."
rm -rf windows linux macos
mkdir -p windows linux

# -----------------------
# Version (Kaskadierendes Increment)
# -----------------------
if [[ -f "$VERSION_FILE" ]]; then
    VERSION=$(cat "$VERSION_FILE")
else
    VERSION="1.0.0"
fi

IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"
PATCH=$((PATCH + 1))

if [ "$PATCH" -gt 999 ]; then
    PATCH=0
    MINOR=$((MINOR + 1))
fi

if [ "$MINOR" -gt 9 ]; then
    MINOR=0
    MAJOR=$((MAJOR + 1))
fi

NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "$NEW_VERSION" > "$VERSION_FILE"

# --- WICHTIG: Hier werden die Variablen für Go definiert ---
# -X setzt die Variable im Go-Code (Paket main, Variable Version)
# -s -w reduziert die Dateigröße (entfernt Debug-Symbole)
GO_LDFLAGS="-X main.Version=${NEW_VERSION} -s -w"

echo "Building Version: $NEW_VERSION"

# -----------------------
# Linux build
# -----------------------
echo "Building Linux/amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$GO_LDFLAGS" -o ./linux/${PRODUCT_NAME} .

# Hash-Datei im Linux-Ordner erstellen
(cd linux && sha256sum ${PRODUCT_NAME} > ${PRODUCT_NAME}.sha256)

# -----------------------
# Windows build
# -----------------------
echo "Building Windows/amd64..."

if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
    echo "Found MinGW, using for Windows CGO build..."
    export CC=x86_64-w64-mingw32-gcc
    export CXX=x86_64-w64-mingw32-g++
    export CGO_ENABLED=1
else
    echo "MinGW not found! Windows build will disable CGO."
    export CGO_ENABLED=0
fi

GOOS=windows GOARCH=amd64 go build -ldflags "$GO_LDFLAGS" -o ./windows/${PRODUCT_NAME}.exe .

# Hash-Datei im Windows-Ordner erstellen
(cd windows && sha256sum ${PRODUCT_NAME}.exe > ${PRODUCT_NAME}.exe.sha256)

# -----------------------
# macOS build
echo "Skipping macOS build on Linux host. Build manually on macOS when needed."

# -----------------------
# Finaler Report
# -----------------------
echo ""
echo "=========================================="
# ... (nach dem Windows & Linux Build) ...

# Hashes extrahieren
LINUX_HASH=$(cat linux/${PRODUCT_NAME}.sha256 | awk '{print $1}')
WIN_HASH=$(cat windows/${PRODUCT_NAME}.exe.sha256 | awk '{print $1}')

rm -Rf linux/${PRODUCT_NAME}.sha256  windows/${PRODUCT_NAME}.exe.sha256

# -----------------------
# HTML Snippet generieren
# -----------------------
cat <<EOF > build_info.html
# -----------------------
# HTML AUTO-UPDATE (Styled)
# -----------------------
cat <<EOF > temp_release.html
<section class="release-container">
    <div class="release-header">
        <span class="release-icon"></span> 
        vbmini Release v$NEW_VERSION
    </div>
    <div class="release-body">
        <div class="release-row">
            <div class="rel-platform"><strong>Linux</strong> (amd64)</div>
            <div class="rel-action"><a href="linux/vbmini" class="download-link">Download</a></div>
            <div class="rel-hash"><code>$LINUX_HASH</code> <small>sha256</small></div>
        </div>
        <div class="release-row">
            <div class="rel-platform"><strong>Windows</strong> (amd64)</div>
            <div class="rel-action"><a href="windows/vbmini.exe" class="download-link">Download</a></div>
            <div class="rel-hash"><code>$WIN_HASH</code> <small>sha256</small></div>
        </div>
    </div>
</section>
EOF

echo "Die Datei 'build_info.html' wurde erfolgreich aktualisiert!"