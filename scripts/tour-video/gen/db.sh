#!/bin/sh
i=0
echo "LOG:  database system is ready to accept connections"
while true; do
  i=$((i+1))
  if [ $((i%6)) -eq 0 ]; then echo "$(date -u '+%F %T UTC') [$((100+i))] LOG:  checkpoint complete: wrote $((i*7%300)) buffers"; 
  elif [ $((i%13)) -eq 0 ]; then echo "$(date -u '+%F %T UTC') [$((100+i))] WARNING:  duration: 1$((i%9))42 ms  statement: SELECT * FROM orders WHERE status='pending'"; 
  else echo "$(date -u '+%F %T UTC') [$((100+i))] LOG:  connection authorized: user=app database=shop"; fi
  sleep 1.$((i%5))
done
