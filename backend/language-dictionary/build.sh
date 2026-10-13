#!/usr/bin/env bash
# Builds the offline English–Vietnamese dictionary of the Reading step,
# data/english-dictionary.db next to this script, from two sources it downloads and verifies:
#   - minhqnd/dictionary v2.0.0 (MIT, built from Wiktionary), ~180 MB;
#   - Hồ Ngọc Đức's Free Vietnamese Dictionary Project, anhviet109K.txt (GPL), ~15 MB.
# The merge (dictmerge/) keeps English words with Vietnamese meanings only, adds the second
# source's meanings that are not already given, and needs Go. The sources are downloaded into
# data/downloads/ and deleted once the merge succeeds (after a failed run they are kept, and the
# next run does not download them again). data/ is ignored by git.
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
DATA="$HERE/data"
SRC="$DATA/downloads"
OUT="$DATA/english-dictionary.db"

MINHQND_URL="https://github.com/minhqnd/dictionary/releases/download/v2.0.0/dictionary.db"
MINHQND_SHA256="9259403f0675b2991a1bd0ef6d0dbc5933afdb135632af095a60662f09bbf1d3"
HND_URL="https://raw.githubusercontent.com/yenthanh132/avdict-database-sqlite-converter/4a7abd2db3626a0fe5091f9db9fa5c75d4014b1f/anhviet109K.txt"
HND_SHA256="e12f69b2184bd782c249ab07cf29b8d5021fbf6237dd29b1e1ff5421e9ea1e2d"

checksum() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1
  else shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

# fetch URL SHA256 FILE: downloads FILE unless it is already there with the right checksum.
fetch() {
  local url="$1" sum="$2" file="$3"
  if [ -f "$file" ] && [ "$(checksum "$file")" = "$sum" ]; then
    echo "Already present: $file"
    return
  fi
  echo "Downloading $url ..."
  curl -fL --retry 3 -o "$file.part" "$url"
  if [ "$(checksum "$file.part")" != "$sum" ]; then
    rm -f "$file.part"
    echo "Checksum mismatch for $url, download removed." >&2
    exit 1
  fi
  mv "$file.part" "$file"
}

mkdir -p "$SRC"
MINHQND="$SRC/source-wiktionary.db"
fetch "$MINHQND_URL" "$MINHQND_SHA256" "$MINHQND"
HND="$SRC/source-ho-ngoc-duc.txt"
fetch "$HND_URL" "$HND_SHA256" "$HND"

echo "Merging..."
(cd "$HERE/.." && go run ./language-dictionary/dictmerge \
  -minhqnd "$MINHQND" \
  -hnd "$HND" \
  -out "$OUT")

rm -rf "$SRC"
echo "Dictionary ready: $OUT (downloaded sources removed)"
echo "The backend reads it by default (DICTIONARY_PATH=./language-dictionary/data/english-dictionary.db);"
echo "restart the backend to use it."
