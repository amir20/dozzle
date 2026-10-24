#!/bin/sh
i=0
while true; do
  i=$((i+1))
  st=200; [ $((i%11)) -eq 0 ] && st=404; [ $((i%29)) -eq 0 ] && st=502
  echo "10.0.$((i%4)).$((i%200+10)) - - [$(date -u '+%d/%b/%Y:%H:%M:%S +0000')] \"GET /$( [ $((i%3)) -eq 0 ] && echo assets/app.js || echo index.html ) HTTP/1.1\" $st $((i*31%9000+512)) \"-\" \"Mozilla/5.0\""
  sleep 0.$(( (i*7) % 5 + 3 ))
done
