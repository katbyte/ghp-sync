#!/bin/sh

# the container is unhealthy from a failed sync until the next one passes; run.sh records how each sync ended

status_file=/var/run/ghp-sync.status

# without cron no sync will ever run again, so the last recorded result would stay "passed" forever
if ! pgrep crond > /dev/null; then
  echo "crond is not running"
  exit 1
fi

# no status file means no sync has finished since the container was created
if [ ! -f "$status_file" ]; then
  echo "no sync has finished yet"
  exit 0
fi

status=$(cat "$status_file")
if [ "$status" != "0" ]; then
  echo "last sync failed with exit code $status, finished $(date -r "$status_file")"
  exit 1
fi

echo "last sync passed, finished $(date -r "$status_file")"
