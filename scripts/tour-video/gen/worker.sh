#!/bin/sh
i=0
while true; do
  i=$((i+1))
  if [ $((i % 17)) -eq 0 ]; then
    echo "$(date -u +%FT%TZ) ERROR job email.send failed: connection refused"
    echo "Traceback (most recent call last):"
    echo "  File \"/app/worker/jobs.py\", line 88, in run"
    echo "    smtp.send(message)"
    echo "  File \"/app/worker/mail.py\", line 41, in send"
    echo "    raise ConnectionRefusedError(\"smtp:587\")"
    echo "ConnectionRefusedError: smtp:587"
  elif [ $((i % 5)) -eq 0 ]; then
    echo "$(date -u +%FT%TZ) WARN queue depth 1$((i%9))4 above threshold"
  else
    echo "$(date -u +%FT%TZ) INFO processed job invoice.render id=$((4000+i)) in $(( (i*13)%90+10 ))ms"
  fi
  sleep 0.$(( (i*5) % 7 + 2 ))
done
