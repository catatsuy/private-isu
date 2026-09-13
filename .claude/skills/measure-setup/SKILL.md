---
name: measure-setup
description: Web アプリの負荷計測環境を構築する。alp と pt-query-digest を導入し、nginx の LTSV アクセスログと MySQL の全クエリ slow log を有効化し、計測スクリプト（docker stats サンプラーと一括実行）を配置する。「計測環境を整えて」「alp を入れて」「slow log を有効化して」「計測の準備」と言われたとき、またはベンチマークの計測を初めて行う前に使う。
---

# 負荷計測環境のセットアップ

docker compose で動く Web アプリ（リバースプロキシ + アプリ + DB + キャッシュ）の
**コンポーネント単位のボトルネックを数字で切り分けられる状態**を作る。

このファイルだけで完結する。**冪等**なので、途中まで済んでいる環境に再実行してよい。
計測結果の読み方とレポート化は `bottleneck-report` スキルが担当する。

## 前提とする構成

| 役割 | 想定 |
|---|---|
| リバースプロキシ | nginx（公式イメージ） |
| アプリ | 任意の言語。nginx が `proxy_pass` する |
| DB | MySQL 8.0 以降 |
| キャッシュ | memcached（任意） |
| 負荷源 | ベンチマーカー or 負荷ツール |

構成が違う場合は最後の「別リポジトリへの転用」を参照。

## 0. 対象環境の変数を決める

以降のコマンドはこれらを使う。**最初に実測して確定させる**こと。

```bash
cd "$(git rev-parse --show-toplevel)"

# compose ファイルの場所
find . -name 'compose.y*ml' -o -name 'docker-compose.y*ml' | grep -v node_modules

# サービス名とコンテナ名
docker compose -f <compose file> ps --format '{{.Service}}\t{{.Name}}\t{{.State}}'

# ネットワーク名（memcached の stats 取得や負荷試験で使う）
docker network ls

# DB の接続情報（compose の environment から読む）
grep -iE 'MYSQL_|_DB_|DATABASE' <compose file>

# Docker VM の CPU 数 — 計測結果の解釈に必須
docker info --format 'NCPU={{.NCPU}} Mem={{.MemTotal}}'
```

以下では例として次を使う。自分の環境の値に読み替えること。

```bash
COMPOSE=webapp/compose.yml
NET=private-isu_my_network
DBUSER=root DBPASS=root
```

## 1. alp — nginx アクセスログ集計

エンドポイント別に count / sum / avg / p50 / p95 / p99 を出す。**中央値と P95 が取れる数少ない経路の 1 つ。**

**`brew install tkuchiki/alp/alp` は使わない。** tap の `git clone` が GitHub 認証を要求して失敗する
（`fatal: could not read Username for 'https://github.com'`）。

```bash
go install github.com/tkuchiki/alp/cmd/alp@latest
```

インストール先に注意。**asdf などで Go を管理している場合、`GOBIN` は `~/go/bin` ではなく処理系のディレクトリになる。**

```bash
go env GOPATH                      # 例: /Users/xxx/.asdf/installs/golang/1.23.2
ls "$(go env GOPATH)/bin/alp"
export PATH="$(go env GOPATH)/bin:$PATH"
```

`alp -v` / `alp --version` は**存在しないフラグ**でエラーになる。動作確認は `alp ltsv --help`。

Go が無い環境では GitHub Releases からバイナリを取得する。

## 2. pt-query-digest — MySQL slow log 集計

クエリ種別ごとに median / avg / 95% と走査行数を出す。

```bash
brew install percona-toolkit     # core tap なので素直に入る
pt-query-digest --version
```

Linux なら `apt install percona-toolkit` / `yum install percona-toolkit`。

## 3. nginx を LTSV で出力させる

設定ファイルが bind mount されていれば、ホスト側で編集して reload すれば反映される。

> **落とし穴: 公式 nginx イメージの `/var/log/nginx/access.log` は `/dev/stdout` へのシンボリックリンク。**
> このパスに書くとログは `docker logs` に流れてファイルには残らず、あとで `cat` するとブロックして固まる。
> **必ず別パスに出す。**

```nginx
# server ブロックの外（ファイル先頭）に追加
log_format ltsv "time:$time_local"
  "\thost:$remote_addr"
  "\tmethod:$request_method"
  "\turi:$request_uri"
  "\tstatus:$status"
  "\tsize:$body_bytes_sent"
  "\treqtime:$request_time"
  "\tapptime:$upstream_response_time"
  "\tupstream:$upstream_addr"
  "\tcache:$upstream_cache_status";

# server ブロックの中に追加
access_log /var/log/nginx/access_ltsv.log ltsv;
```

ラベル名は alp のデフォルト（`uri` / `method` / `time` / `apptime` / `reqtime` / `size` / `status`）に
合わせてあるので追加指定は不要。変える必要が出たときの**フラグ名は `--reqtime-key` ではなく `--reqtime-label`**。

各フィールドの意味:

| フィールド | 意味 |
|---|---|
| `reqtime` | クライアントから見たレイテンシー |
| `apptime` | アプリ単体のレイテンシー。`reqtime - apptime` がプロキシ自身と転送のコスト |
| `upstream` | `-` ならプロキシが自分で返した（静的ファイル等） |
| `cache` | `HIT` / `MISS` / `-`（対象外）。`proxy_cache` 未設定なら全行 `-` |

反映と確認:

```bash
docker compose -f $COMPOSE exec -T nginx nginx -t
docker compose -f $COMPOSE exec -T nginx nginx -s reload
curl -s -o /dev/null http://localhost/
docker compose -f $COMPOSE exec -T nginx tail -1 /var/log/nginx/access_ltsv.log
```

## 4. MySQL の全クエリ slow log

再起動は不要。`SET GLOBAL` で足りる。`long_query_time = 0` で**全クエリ**が記録される。

```bash
docker compose -f $COMPOSE exec -T mysql mysql -u$DBUSER -p$DBPASS -e "
  SET GLOBAL slow_query_log = 1;
  SET GLOBAL slow_query_log_file = '/var/lib/mysql/slow.log';
  SET GLOBAL long_query_time = 0;
  SET GLOBAL log_slow_extra = 1;"
```

> **落とし穴 1: 確認は `@@global.` を付ける。**
> 同一セッションで `SELECT @@long_query_time` を見ると、接続時に引き継いだ古い値が返って
> 「効いていない」と誤解する。
>
> ```bash
> docker compose -f $COMPOSE exec -T mysql mysql -u$DBUSER -p$DBPASS -N -e \
>   "SELECT @@global.slow_query_log, @@global.long_query_time, @@global.log_slow_extra"
> ```

> **落とし穴 2: `long_query_time` は既存接続に効かない。**
> セッション値は接続時にコピーされるため、アプリが張りっぱなしのコネクションは古い値のまま。
> **設定後にアプリを再起動して接続を貼り直す。**
>
> ```bash
> docker compose -f $COMPOSE restart app && sleep 3
> curl -s -o /dev/null http://localhost/
> docker compose -f $COMPOSE exec -T mysql wc -l /var/lib/mysql/slow.log   # 増えていれば OK
> ```

`SET GLOBAL` は MySQL 再起動で消える。永続化するなら compose の mysql サービスに:

```yaml
    command: --slow_query_log=1 --slow_query_log_file=/var/lib/mysql/slow.log --long_query_time=0 --log_slow_extra=1
```

`performance_schema` が ON なら slow log を触らずに済ませることもできるが、
`events_statements_summary_by_digest` には **median が無い**（`QUANTILE_95` / `QUANTILE_99` はある）。
中央値が要るなら slow log を使う。

## 5. `scripts/sample-stats.sh` — CPU / メモリのサンプラー

`docker stats` を 1 秒ごとに CSV へ落とす。**そのまま貼って `chmod +x`。**

```bash
#!/bin/bash
# docker stats を1秒ごとにCSVへ記録する。usage: ./sample-stats.sh out.csv
# 停止は kill。FILTER で対象コンテナを絞る（既定は全部）。
out=${1:-stats.csv}
FILTER=${FILTER:-.}
echo "ts,name,cpu_pct,mem_used,mem_limit,mem_pct,net_io,block_io" > "$out"
while true; do
  docker stats --no-stream --format '{{.Name}},{{.CPUPerc}},{{.MemUsage}},{{.MemPerc}},{{.NetIO}},{{.BlockIO}}' \
    | grep -E "$FILTER" \
    | sed "s/^/$(date +%s),/; s/ \/ /,/g; s/%//g" >> "$out"
done
```

> **落とし穴: 負荷源のコンテナ名でフィルタから漏れやすい。**
> `docker run` で起動したベンチマーカーはランダム名になる。負荷源の CPU も見たいときは
> `--name bench` を付けて起動し、フィルタに含める。負荷源が同じ Docker VM 内で動く場合、
> その CPU 消費はサーバーと同じ資源を食う。

`docker stats` の CPU% は **1 コア = 100%**。コンテナに `cpus: "1"` の制限があれば 100% が上限。

## 6. `scripts/measure.sh` — 計測一式

ログのリセット → 前スナップショット → サンプラー起動 → ベンチ → 後スナップショット → 回収 → 集計。
**そのまま貼って `chmod +x`。先頭の変数を自分の環境に合わせる。**

```bash
#!/bin/bash
# ベンチ1回ぶんの計測を一括で行う。usage: ./scripts/measure.sh [出力ディレクトリ]
set -eu

# ---- 環境に合わせて変更する ----
COMPOSE=${COMPOSE:-webapp/compose.yml}
NET=${NET:-private-isu_my_network}
DBUSER=${DBUSER:-root}
DBPASS=${DBPASS:-root}
DBNAME=${DBNAME:-isuconp}
APP_SVC=${APP_SVC:-app}
BENCH_CMD=${BENCH_CMD:-"docker run --rm --name bench --network host -i private-isu-benchmarker /bin/benchmarker -t http://host.docker.internal -u /opt/userdata"}
ALP=${ALP:-alp}
# --------------------------------

REPO="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO"
OUT=${1:-$REPO/measure-$(date +%Y%m%d-%H%M%S)}
mkdir -p "$OUT"
dc() { docker compose -f "$COMPOSE" "$@"; }

echo "== reset logs"
dc exec -T nginx sh -c ': > /var/log/nginx/access_ltsv.log'
dc exec -T mysql sh -c ': > /var/lib/mysql/slow.log'
dc exec -T mysql mysql -u$DBUSER -p$DBPASS -e "
  SET GLOBAL slow_query_log=1; SET GLOBAL long_query_time=0; SET GLOBAL log_slow_extra=1;
  TRUNCATE performance_schema.events_statements_summary_by_digest;" 2>/dev/null
# 既存接続は古い long_query_time を引き継ぐため貼り直す
dc restart $APP_SVC >/dev/null 2>&1; sleep 3

echo "== snapshot before"
dc exec -T mysql mysql -u$DBUSER -p$DBPASS -N -e "SHOW GLOBAL STATUS" 2>/dev/null > "$OUT/mysql-before.txt"
docker run --rm --network "$NET" busybox sh -c 'echo stats | nc memcached 11211' > "$OUT/mc-before.txt" 2>/dev/null || true

echo "== start sampler"
"$REPO/scripts/sample-stats.sh" "$OUT/stats.csv" & SAMPLER=$!
trap 'kill $SAMPLER 2>/dev/null || true' EXIT

echo "== bench"
START=$(date +%s)
eval "$BENCH_CMD" | tee "$OUT/bench.json"
ELAPSED=$(( $(date +%s) - START ))
echo "$ELAPSED" > "$OUT/elapsed.txt"
kill $SAMPLER 2>/dev/null || true; trap - EXIT

echo "== snapshot after"
dc exec -T mysql mysql -u$DBUSER -p$DBPASS -N -e "SHOW GLOBAL STATUS" 2>/dev/null > "$OUT/mysql-after.txt"
dc exec -T mysql mysql -u$DBUSER -p$DBPASS -e "
  SELECT LEFT(DIGEST_TEXT,70) AS query, COUNT_STAR AS calls,
    ROUND(SUM_TIMER_WAIT/1e9,1) AS total_ms, ROUND(AVG_TIMER_WAIT/1e9,3) AS avg_ms,
    ROUND(QUANTILE_95/1e9,3) AS p95_ms, SUM_ROWS_EXAMINED AS rows_examined, SUM_ROWS_SENT AS rows_sent
  FROM performance_schema.events_statements_summary_by_digest
  WHERE SCHEMA_NAME='$DBNAME' ORDER BY SUM_TIMER_WAIT DESC LIMIT 20" 2>/dev/null > "$OUT/pfs-digest.txt"
docker run --rm --network "$NET" busybox sh -c 'echo stats | nc memcached 11211' > "$OUT/mc-after.txt" 2>/dev/null || true

echo "== collect logs"
# docker compose cp を使う。exec cat はシンボリックリンク経由でハングしうる
dc cp nginx:/var/log/nginx/access_ltsv.log "$OUT/access.log" >/dev/null
dc cp mysql:/var/lib/mysql/slow.log "$OUT/slow.log" >/dev/null

echo "== aggregate"
"$ALP" ltsv --file "$OUT/access.log" --sort sum -r \
  -m '/posts/[0-9]+,/image/[0-9]+\.(jpg|png|gif),/@[0-9a-zA-Z_]+' \
  --percentiles 50,95 -o count,method,uri,min,max,sum,avg,p50,p95 > "$OUT/alp.txt"
pt-query-digest "$OUT/slow.log" > "$OUT/slow-digest.txt" 2>/dev/null

awk -F, 'NR>1 && $3!="" {c[$2]+=$3; n[$2]++; if($3>cm[$2])cm[$2]=$3; if($6>mm[$2])mm[$2]=$6}
  END {for(k in c) printf "%-26s cpu avg %6.1f%%  cpu max %6.1f%%  mem max %5.1f%%\n", k, c[k]/n[k], cm[k], mm[k]}' \
  "$OUT/stats.csv" | sort > "$OUT/stats-summary.txt"

join "$OUT/mysql-before.txt" "$OUT/mysql-after.txt" 2>/dev/null \
  | awk -v e="$ELAPSED" '$3+0!=$2+0 {printf "%-40s %14d  (%.1f/s)\n", $1, $3-$2, ($3-$2)/e}' > "$OUT/mysql-delta.txt"

echo "== done: $OUT  (elapsed ${ELAPSED}s)"
cat "$OUT/stats-summary.txt"
```

`-m` はパスの ID 部分をまとめる正規表現。**自分のアプリのルーティングに合わせて必ず書き換える。**
これを忘れると `/posts/1` `/posts/2` が別行に散って集計が意味を成さない。

## 7. 最終確認

```bash
cd "$(git rev-parse --show-toplevel)"
export PATH="$(go env GOPATH)/bin:$PATH"
alp ltsv --help >/dev/null && echo "alp OK"
pt-query-digest --version >/dev/null && echo "pt-query-digest OK"
docker compose -f $COMPOSE exec -T nginx test -f /var/log/nginx/access_ltsv.log && echo "nginx ltsv OK"
docker compose -f $COMPOSE exec -T mysql mysql -u$DBUSER -p$DBPASS -N -e "SELECT @@global.long_query_time" | grep -q '^0' && echo "slow log OK"
ls scripts/measure.sh scripts/sample-stats.sh >/dev/null && echo "scripts OK"
```

5 つとも OK なら `./scripts/measure.sh` を実行できる。

## 環境依存の落とし穴

**`docker info` の `NCPU` を必ず確認する。**
Docker Desktop の VM は既定で CPU 割り当てが少ない（実測例: ホスト 11 コアに対し VM は 2）。
compose が app と DB にそれぞれ `cpus: "1"` を与えていると、この 2 つだけで VM の全量になる。
この状態ではどのコンテナも 100% に張り付かないまま頭打ちになり、計測の解釈を誤らせる。
負荷源も同じ VM 内で動かしていれば、さらにその分を食う。

対処は Docker Desktop の Settings → Resources で CPU を増やすか、負荷源を VM の外で動かす。

## 別リポジトリへの転用

| このスキルの前提 | 違う場合 |
|---|---|
| nginx | Apache なら `LogFormat` で同等の LTSV を出す。Envoy/Traefik はアクセスログの JSON を `alp json` で食わせる |
| MySQL | PostgreSQL なら `log_min_duration_statement = 0` + `pgbadger`。`auto_explain` も有効 |
| memcached | Redis なら `redis-cli INFO stats` の `keyspace_hits` / `keyspace_misses` |
| docker compose | 素の Docker なら `docker cp` / `docker exec`。VM や実機なら `dstat` / `sar` に置き換える |
| ベンチマーカー | `BENCH_CMD` を差し替える。`hey` / `k6` / `wrk` でもよい |

変わらないのは **「アクセスログで p50/p95 を、DB のクエリログで走査行数を、`docker stats` で CPU 飽和を見る」** という三点測量の構造。
この 3 つが揃えば、どの構成でもコンポーネント単位の切り分けはできる。
