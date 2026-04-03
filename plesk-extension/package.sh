#!/usr/bin/env bash
# Packages the extension into a ZIP file ready for Plesk upload.
# Usage: ./package.sh
set -euo pipefail

NAME="automation-hub"
VERSION=$(grep -oP '(?<=<version>)[^<]+' meta.xml)
OUT="${NAME}-${VERSION}.zip"

rm -f "$OUT"
zip -r "$OUT" meta.xml plib/ htdocs/ --exclude "*.DS_Store" --exclude "__pycache__"
echo "Created: $OUT"
echo "Install via: Plesk → Extensions → Upload Extension"
