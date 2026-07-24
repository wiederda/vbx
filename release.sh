#!/bin/bash
set -e

CONFIG_FILE="release.json"
VERSION_FILE="build_number.txt"

# -----------------------
# Voraussetzungen
# -----------------------

for CMD in jq curl; do
    if ! command -v "$CMD" >/dev/null 2>&1; then
        echo "Fehler: $CMD nicht installiert"
        exit 1
    fi
done


if [ ! -f "$CONFIG_FILE" ]; then
    echo "Fehler: $CONFIG_FILE fehlt"
    exit 1
fi


# -----------------------
# GitHub Konfiguration
# -----------------------

GITHUB_TOKEN=$(jq -r '.github.token' "$CONFIG_FILE")
GITHUB_REPO=$(jq -r '.github.repo' "$CONFIG_FILE")
GITHUB_API=$(jq -r '.github.api' "$CONFIG_FILE")

RELEASE_TITLE=$(jq -r '.release.title' "$CONFIG_FILE")
RELEASE_DESC=$(jq -r '.release.description' "$CONFIG_FILE")


if [ "$GITHUB_TOKEN" = "null" ] || [ -z "$GITHUB_TOKEN" ]; then
    echo "Fehler: GitHub Token fehlt"
    exit 1
fi

if [ "$GITHUB_REPO" = "null" ] || [ -z "$GITHUB_REPO" ]; then
    echo "Fehler: GitHub Repository fehlt"
    exit 1
fi


echo "Repository: $GITHUB_REPO"


# -----------------------
# Build ausführen
# -----------------------

echo "Building..."

./build.sh


# -----------------------
# Version lesen
# -----------------------

if [ ! -f "$VERSION_FILE" ]; then
    echo "Fehler: $VERSION_FILE fehlt"
    exit 1
fi


VERSION=$(cat "$VERSION_FILE")
TAG="v${VERSION}"


echo "Release Version: $TAG"


# -----------------------
# GitHub Release erstellen
# -----------------------

echo "Creating GitHub release..."


RESPONSE=$(curl \
    -s \
    -X POST \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "${GITHUB_API}/repos/${GITHUB_REPO}/releases" \
    -d "{
        \"tag_name\":\"${TAG}\",
        \"name\":\"${RELEASE_TITLE} ${TAG}\",
        \"body\":\"${RELEASE_DESC}\"
    }")


UPLOAD_URL=$(echo "$RESPONSE" \
    | jq -r '.upload_url' \
    | sed 's/{?name,label}//')


if [ "$UPLOAD_URL" = "null" ] || [ -z "$UPLOAD_URL" ]; then
    echo "Fehler beim Erstellen des Releases:"
    echo "$RESPONSE"
    exit 1
fi


# -----------------------
# Upload Funktion
# -----------------------

upload_asset()
{
    FILE="$1"

    if [ ! -f "$FILE" ]; then
        echo "Datei fehlt: $FILE"
        exit 1
    fi

    echo "Upload: $FILE"

    curl \
        -s \
        -X POST \
        -H "Authorization: Bearer ${GITHUB_TOKEN}" \
        -H "Content-Type: application/octet-stream" \
        --data-binary @"$FILE" \
        "${UPLOAD_URL}?name=$(basename "$FILE")"
}


# -----------------------
# Release Dateien hochladen
# -----------------------

upload_asset "linux/vbx"
upload_asset "windows/vbx.exe"


# -----------------------
# Fertig
# -----------------------

echo ""
echo "================================"
echo "Release ${TAG} erfolgreich"
echo "================================"