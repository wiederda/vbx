#!/bin/bash
set -e

echo "Creating VBX release..."

# Build ausführen
./build.sh

# aktuelle Version lesen
VERSION=$(cat build_number.txt)

echo "Release version: $VERSION"

# Git Tag erstellen
git add .
git commit -m "Release v$VERSION"
git tag "v$VERSION"
git push
git push origin "v$VERSION"

# GitHub Release erzeugen
gh release create "v$VERSION" \
    linux/vbx \
    windows/vbx.exe \
    --title "VBX v$VERSION" \
    --notes "Release v$VERSION"