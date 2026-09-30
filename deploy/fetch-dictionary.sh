#!/usr/bin/env bash
# Downloads the offline English–Vietnamese dictionary (minhqnd/dictionary v2.0.0, MIT) used by
# the Reading step, into deploy/data/dictionary/dictionary.db, and verifies its SHA-256.
set -euo pipefail

URL="https://github.com/minhqnd/dictionary/releases/download/v2.0.0/dictionary.db"
SHA256="9259403f0675b2991a1bd0ef6d0dbc5933afdb135632af095a60662f09bbf1d3"
DIR="$(cd "$(dirname "$0")" && pwd)/data/dictionary"
FILE="$DIR/dictionary.db"

checksum() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1
  else shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

mkdir -p "$DIR"
if [ -f "$FILE" ] && [ "$(checksum "$FILE")" = "$SHA256" ]; then
  echo "Dictionary already present: $FILE"
  exit 0
fi

echo "Downloading dictionary (~180 MB)..."
curl -fL --retry 3 -o "$FILE.part" "$URL"
if [ "$(checksum "$FILE.part")" != "$SHA256" ]; then
  rm -f "$FILE.part"
  echo "Checksum mismatch, download removed." >&2
  exit 1
fi
mv "$FILE.part" "$FILE"
echo "Dictionary ready: $FILE"
