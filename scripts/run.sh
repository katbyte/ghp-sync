#!/bin/sh

echo
echo "Job started: $(date)"
# shellcheck disable=SC2086 # SYNC_CMD is intentionally unquoted so subcommand arguments split, e.g. "prs refresh"
ghp-sync $SYNC_CMD
status=$?
echo "Job finished: $(date)"

# record how the sync ended for the healthcheck; written aside and moved into place so a check never reads a half-written file
echo "$status" > /var/run/ghp-sync.status.tmp
mv /var/run/ghp-sync.status.tmp /var/run/ghp-sync.status
