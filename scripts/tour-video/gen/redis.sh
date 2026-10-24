#!/bin/sh
i=0
while true; do i=$((i+1)); echo "1:M $(date -u '+%d %b %Y %H:%M:%S').000 * $((i%50+1)) changes in 60 seconds. Saving..."; echo "1:M $(date -u '+%d %b %Y %H:%M:%S').010 * Background saving terminated with success"; sleep 2; done
