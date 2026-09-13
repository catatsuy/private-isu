#!/bin/bash
# ベンチ1回ぶんの計測を一括で行う。usage: ./scripts/measure.sh [出力ディレクトリ]
set -eu
ALP=${ALP:-alp}
REPO="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO/webapp"
OUT=${1:-$REPO/measure-$(date +%Y%m%d-%H%M%S)}
mkdir -p "$OUT"

echo "== reset logs"
docker compose exec -T nginx sh -c ': > /var/log/nginx/access_ltsv.log'
docker compose exec -T mysql sh -c ': > /var/lib/mysql/slow.log'
docker compose exec -T mysql mysql -uroot -proot -e "SET GLOBAL slow_query_log=1; SET GLOBAL long_query_time=0; SET GLOBAL log_slow_extra=1" 2>/dev/null
docker compose restart app >/dev/null 2>&1; sleep 3  # 既存接続は古い long_query_time を引き継ぐため貼り直す
docker compose exec -T mysql mysql -uroot -proot -e "TRUNCATE performance_schema.events_statements_summary_by_digest" 2>/dev/null

echo "== snapshot before"
docker compose exec -T mysql mysql -uroot -proot -N -e "SHOW GLOBAL STATUS" 2>/dev/null > "$OUT/mysql-before.txt"
docker run --rm --network private-isu_my_network busybox sh -c 'echo stats | nc memcached 11211' > "$OUT/mc-before.txt"

echo "== start docker stats sampler"
"$REPO/scripts/sample-stats.sh" "$OUT/stats.csv" & SAMPLER=$!
trap 'kill $SAMPLER 2>/dev/null || true' EXIT

echo "== bench"
START=$(date +%s)
docker run --network host -i private-isu-benchmarker /bin/benchmarker \
  -t http://host.docker.internal -u /opt/userdata | tee "$OUT/bench.json"
ELAPSED=$(( $(date +%s) - START ))
echo "$ELAPSED" > "$OUT/elapsed.txt"

kill $SAMPLER 2>/dev/null || true
trap - EXIT

echo "== snapshot after"
docker compose exec -T mysql mysql -uroot -proot -N -e "SHOW GLOBAL STATUS" 2>/dev/null > "$OUT/mysql-after.txt"
docker compose exec -T mysql mysql -uroot -proot -e "
  SELECT LEFT(DIGEST_TEXT,70) AS query, COUNT_STAR AS calls,
    ROUND(SUM_TIMER_WAIT/1e9,1) AS total_ms, ROUND(AVG_TIMER_WAIT/1e9,3) AS avg_ms,
    ROUND(QUANTILE_95/1e9,3) AS p95_ms, SUM_ROWS_EXAMINED AS rows_examined, SUM_ROWS_SENT AS rows_sent
  FROM performance_schema.events_statements_summary_by_digest
  WHERE SCHEMA_NAME='isuconp' ORDER BY SUM_TIMER_WAIT DESC LIMIT 20" 2>/dev/null > "$OUT/pfs-digest.txt"
docker run --rm --network private-isu_my_network busybox sh -c 'echo stats | nc memcached 11211' > "$OUT/mc-after.txt"

echo "== collect logs"
docker compose cp nginx:/var/log/nginx/access_ltsv.log "$OUT/access.log" >/dev/null
docker compose cp mysql:/var/lib/mysql/slow.log "$OUT/slow.log" >/dev/null

echo "== aggregate"
"$ALP" ltsv --file "$OUT/access.log" --sort sum -r \
  -m '/posts/[0-9]+,/image/[0-9]+\.(jpg|png|gif),/@[0-9a-zA-Z_]+' \
  --percentiles 50,95 -o count,method,uri,min,max,sum,avg,p50,p95 > "$OUT/alp.txt"
pt-query-digest "$OUT/slow.log" > "$OUT/slow-digest.txt" 2>/dev/null

awk -F, 'NR>1 {c[$2]+=$3; n[$2]++; if($3>cm[$2])cm[$2]=$3; if($6>mm[$2])mm[$2]=$6}
  END {for(k in c) printf "%-26s cpu avg %6.1f%%  cpu max %6.1f%%  mem max %5.1f%%\n", k, c[k]/n[k], cm[k], mm[k]}' \
  "$OUT/stats.csv" | sort > "$OUT/stats-summary.txt"

# GLOBAL STATUS 差分
join "$OUT/mysql-before.txt" "$OUT/mysql-after.txt" 2>/dev/null \
  | awk -v e="$ELAPSED" '$3+0!=$2+0 {printf "%-40s %14d  (%.1f/s)\n", $1, $3-$2, ($3-$2)/e}' > "$OUT/mysql-delta.txt"

# memcached 差分
join -j2 <(grep ^STAT "$OUT/mc-before.txt" | sort -k2) <(grep ^STAT "$OUT/mc-after.txt" | sort -k2) 2>/dev/null \
  | awk -v e="$ELAPSED" '$5 ~ /^[0-9]+$/ && $3 ~ /^[0-9]+$/ && $5-$3 != 0 {printf "%-24s %12d  (%.1f/s)\n", $1, $5-$3, ($5-$3)/e}' > "$OUT/mc-delta.txt"

echo "== done: $OUT  (elapsed ${ELAPSED}s)"
