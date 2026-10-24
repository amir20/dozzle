#!/bin/sh
i=0
paths="/api/cart /api/checkout /api/products /api/orders /health /api/users/me /api/search"
while true; do
  i=$((i+1))
  p=$(echo $paths | tr ' ' '\n' | sed -n "$(( i % 7 + 1 ))p")
  ms=$(( (i*37) % 180 + 4 ))
  if [ $((i % 23)) -eq 0 ]; then
    echo "{\"level\":\"error\",\"msg\":\"payment provider timeout\",\"path\":\"/api/checkout\",\"status\":504,\"duration_ms\":3001,\"order_id\":\"ord_$((8000+i))\"}"
  elif [ $((i % 9)) -eq 0 ]; then
    echo "{\"level\":\"warn\",\"msg\":\"slow query\",\"table\":\"orders\",\"duration_ms\":$((ms*6))}"
  else
    echo "{\"level\":\"info\",\"msg\":\"request completed\",\"method\":\"GET\",\"path\":\"$p\",\"status\":200,\"duration_ms\":$ms}"
  fi
  sleep 0.$(( (i*3) % 6 + 2 ))
done
