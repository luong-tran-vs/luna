#!/usr/bin/env bash
# Daily MongoDB backup for Luna (F13). Runs in the "backup" service (image mongo:8).
#
# Every day at BACKUP_TIME (in TZ) it writes /backups/luna-YYYYMMDD.archive.gz with
# mongodump --archive --gzip, then keeps the newest BACKUP_KEEP backups. A failed backup is
# logged, deletes nothing and never stops the loop; the next run tries again.
#
#   BACKUP_NOW=1   back up when the service starts, then follow the schedule
#   BACKUP_ONCE=1  back up once and exit with its result (no schedule)
set -uo pipefail

MONGO_URI="${MONGO_URI:-mongodb://mongo:27017}"
MONGO_DATABASE="${MONGO_DATABASE:-luna}"
BACKUP_DIR="${BACKUP_DIR:-/backups}"
BACKUP_TIME="${BACKUP_TIME:-03:00}"
BACKUP_KEEP="${BACKUP_KEEP:-7}"
BACKUP_NOW="${BACKUP_NOW:-0}"
BACKUP_ONCE="${BACKUP_ONCE:-0}"

log() {
  local level="$1"
  shift
  echo "$(date --iso-8601=seconds) ${level} $*"
}

backup_name() {
  echo "luna-$(date +%Y%m%d).archive.gz"
}

# rotate deletes all but the newest BACKUP_KEEP backups. Only files named like a backup are
# considered, so anything else in the directory is left alone.
rotate() {
  local files=()
  mapfile -t files < <(find "$BACKUP_DIR" -maxdepth 1 -type f -regextype posix-extended -regex '.*/luna-[0-9]{8}\.archive\.gz' -printf '%f\n' | sort)
  local extra=$(( ${#files[@]} - BACKUP_KEEP ))
  local i
  for (( i = 0; i < extra; i++ )); do
    if rm -f -- "$BACKUP_DIR/${files[$i]}"; then
      log INFO "deleted old backup ${files[$i]}"
    else
      log ERROR "could not delete old backup ${files[$i]}"
    fi
  done
}

# run_backup writes today's backup to a temporary file and renames it only when mongodump
# succeeded, so a failed or interrupted run never replaces a good backup or triggers rotation.
run_backup() {
  local name tmp started output size
  name="$(backup_name)"
  tmp="$BACKUP_DIR/$name.tmp"
  started=$(date +%s)
  log INFO "backup started: $name"
  if ! output=$(mongodump --uri="$MONGO_URI" --db="$MONGO_DATABASE" --archive="$tmp" --gzip 2>&1); then
    rm -f -- "$tmp"
    log ERROR "backup failed: $(echo "$output" | tail -n 3 | tr '\n' ' ')"
    return 1
  fi
  if [[ ! -s "$tmp" ]]; then
    rm -f -- "$tmp"
    log ERROR "backup failed: mongodump wrote an empty file"
    return 1
  fi
  if ! mv -f -- "$tmp" "$BACKUP_DIR/$name"; then
    rm -f -- "$tmp"
    log ERROR "backup failed: could not write $BACKUP_DIR/$name"
    return 1
  fi
  size=$(du -h "$BACKUP_DIR/$name" | cut -f1)
  log INFO "backup done: $name ($size, $(( $(date +%s) - started )) s)"
  rotate
}

# next_run prints the epoch seconds of the next BACKUP_TIME.
next_run() {
  local today
  today=$(date -d "today $BACKUP_TIME" +%s)
  if (( today > $(date +%s) )); then
    echo "$today"
  else
    date -d "tomorrow $BACKUP_TIME" +%s
  fi
}

if ! [[ "$BACKUP_TIME" =~ ^([01][0-9]|2[0-3]):[0-5][0-9]$ ]]; then
  log ERROR "BACKUP_TIME must be HH:MM, got '$BACKUP_TIME'"
  exit 1
fi
if ! [[ "$BACKUP_KEEP" =~ ^[1-9][0-9]*$ ]]; then
  log ERROR "BACKUP_KEEP must be a positive number, got '$BACKUP_KEEP'"
  exit 1
fi
if ! mkdir -p "$BACKUP_DIR"; then
  log ERROR "cannot create $BACKUP_DIR"
  exit 1
fi

# Leftovers of an interrupted run are never kept.
find "$BACKUP_DIR" -maxdepth 1 -type f -name 'luna-*.archive.gz.tmp' -delete

if [[ "$BACKUP_ONCE" == "1" ]]; then
  run_backup
  exit $?
fi

trap 'log INFO "backup service stopping"; exit 0' TERM INT

# Back up now when asked, or when today's time has passed without today's backup (the service
# was down at BACKUP_TIME).
if [[ "$BACKUP_NOW" == "1" ]]; then
  run_backup
elif (( $(date -d "today $BACKUP_TIME" +%s) <= $(date +%s) )) && [[ ! -f "$BACKUP_DIR/$(backup_name)" ]]; then
  log INFO "today's backup is missing, backing up now"
  run_backup
fi

while true; do
  next=$(next_run)
  log INFO "next backup at $(date -d "@$next" --iso-8601=seconds)"
  # sleep in the background so SIGTERM stops the service at once.
  sleep $(( next - $(date +%s) )) &
  wait $!
  run_backup
done
