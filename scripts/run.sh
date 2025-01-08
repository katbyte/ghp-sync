#!/bin/sh

echo
echo "Job started: $(date)"
ghp-sync "$SYNC_CMD"
echo "Job finished: $(date)"
