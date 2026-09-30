#!/usr/bin/env bash
# Restores a Luna backup (F13). Run from the repository root:
#
#   docker compose -f deploy/docker-compose.yml stop backend
#   docker compose -f deploy/docker-compose.yml run --rm --entrypoint bash backup /backup/restore.sh luna-20260930.archive.gz
#   docker compose -f deploy/docker-compose.yml start backend
#
# Every collection in the backup REPLACES the current one (mongorestore --drop), including
# accounts and login sessions. The file is checked first: a missing or corrupt backup changes
# nothing.
set -uo pipefail

MONGO_URI="${MONGO_URI:-mongodb://mongo:27017}"
MONGO_DATABASE="${MONGO_DATABASE:-luna}"
BACKUP_DIR="${BACKUP_DIR:-/backups}"

list_backups() {
  echo "Available backups in $BACKUP_DIR:"
  local found=0 f
  for f in "$BACKUP_DIR"/luna-*.archive.gz; do
    [[ -f "$f" ]] || continue
    found=1
    echo "  $(basename "$f")  ($(du -h "$f" | cut -f1))"
  done
  if (( found == 0 )); then
    echo "  (none)"
  fi
}

if [[ $# -ne 1 ]]; then
  echo "Usage: restore.sh <backup file>   e.g. restore.sh luna-20260930.archive.gz"
  list_backups
  exit 1
fi

file="$1"
# A bare name refers to the backup directory.
if [[ "$file" != */* ]]; then
  file="$BACKUP_DIR/$file"
fi
if [[ ! -f "$file" ]]; then
  echo "ERROR backup not found: $1"
  list_backups
  exit 1
fi
if ! gzip -t "$file" 2>/dev/null; then
  echo "ERROR backup is corrupt: $1 (the database was not changed)"
  exit 1
fi

echo "Restoring $(basename "$file") into database '$MONGO_DATABASE' (current data is replaced)..."
mongorestore --uri="$MONGO_URI" --archive="$file" --gzip --drop --nsInclude="$MONGO_DATABASE.*"
status=$?
if (( status != 0 )); then
  echo "ERROR mongorestore failed (exit $status)"
  exit "$status"
fi

echo "Restore done. Documents per collection:"
mongosh "$MONGO_URI/$MONGO_DATABASE" --quiet --eval \
  'db.getCollectionNames().sort().forEach(c => print("  " + c + ": " + db.getCollection(c).countDocuments()))'
