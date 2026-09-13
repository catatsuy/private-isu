#!/bin/bash
# docker stats を1秒ごとにCSVへ記録する。usage: ./sample-stats.sh out.csv
out=${1:-stats.csv}
echo "ts,name,cpu_pct,mem_used,mem_limit,mem_pct,net_io,block_io" > "$out"
while true; do
  docker stats --no-stream --format '{{.Name}},{{.CPUPerc}},{{.MemUsage}},{{.MemPerc}},{{.NetIO}},{{.BlockIO}}' \
    | grep private-isu \
    | sed "s/^/$(date +%s),/; s/ \/ /,/g; s/%//g" >> "$out"
done
