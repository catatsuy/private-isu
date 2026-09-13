---
name: bottleneck-report
description: 負荷試験を計測し、どのコンポーネント（プロキシ / アプリ / DB / キャッシュ）がボトルネックかを数字で切り分けてレポートにまとめる。「ボトルネックを調査して」「計測して」「ベンチ回して結果をまとめて」「改善の効果を測って」「パフォーマンスを分析して」と言われたときに使う。計測の実行、数字を読む順番、飽和判定、結論の出し方、レポート構成を含む。
---

# ボトルネック計測とレポート作成

このファイルだけで完結する。計測環境が未整備なら先に `measure-setup` スキルで
alp / pt-query-digest の導入と、nginx の LTSV ログ・DB の全クエリログを有効化しておく。

## 原則

1. **推測を書かない。** 「おそらく N+1 が原因」ではなく「pt-query-digest で 65.3%」と書く。数字が無い主張は載せない。
2. **1 回の改善で 1 つだけ変える。** 複数同時に入れると、どれが効いたか永久に分からなくなる。
3. **改善のたびに測り直す。** ボトルネックは動く。前回無関係だったものが次で 1 位になる。
4. **効果が無かったことも記録する。** 次に同じ手を試すのを防げる。
5. **「一番遅いもの」ではなく「合計時間が一番大きいもの」を直す。** 1 件 5 秒のエンドポイントより、10ms × 2 万件のほうが大きいことがある。

## 計測の実行

```bash
cd "$(git rev-parse --show-toplevel)"
export PATH="$(go env GOPATH)/bin:$PATH"      # alp が PATH に無い場合
./scripts/measure.sh measure-$(date +%Y%m%d-%H%M%S)
```

出力ディレクトリ（以降 `$M`）に揃うもの:

| ファイル | 中身 |
|---|---|
| `bench.json` | ベンチの結果（スコア等） |
| `elapsed.txt` | 計測秒数 |
| `stats-summary.txt` | コンテナごとの CPU avg/max、メモリ max |
| `stats.csv` | `docker stats` の生データ |
| `access.log` | nginx の LTSV アクセスログ |
| `alp.txt` | エンドポイント別の count / sum / avg / p50 / p95 |
| `slow.log` | MySQL の全クエリログ |
| `slow-digest.txt` | pt-query-digest の出力 |
| `pfs-digest.txt` | performance_schema のダイジェスト |
| `mysql-{before,after}.txt` | `SHOW GLOBAL STATUS` |
| `mysql-delta.txt` | その差分 |
| `mc-{before,after}.txt` | memcached の `stats` |

## 数字を読む順番

**この順で見る。** 飛ばすと効かない場所を最適化することになる。

### 1. どのコンポーネントが飽和しているか

```bash
cat $M/stats-summary.txt
```

```
private-isu-app-1          cpu avg   51.8%  cpu max   64.5%  mem max   6.9%
private-isu-mysql-1        cpu avg   92.1%  cpu max  104.3%  mem max  65.2%
```

compose で `cpus: "1"` の制限があれば **100% が上限**。ここで 100% 近いものが第一容疑者。
**どれも 100% に届いていないときは、結論を書く前に後述の「飽和判定」へ進む。**

### 2. 時間がプロキシ / アプリ / DB のどこで消えているか

```bash
awk -F'\t' '{for(i=1;i<=NF;i++){split($i,a,":"); v[a[1]]=substr($i,length(a[1])+2)}
  r+=v["reqtime"]; if(v["apptime"]!="-"&&v["apptime"]!="")ap+=v["apptime"]; n++}
  END {printf "requests %d  reqtime %.1fs  apptime %.1fs  diff %.1fs (%.1f%%)\n", n,r,ap,r-ap,(r-ap)/r*100}' $M/access.log

# プロキシが自分で返した割合（静的ファイルの肩代わり状況）
awk -F'\t' '{if($0 ~ /upstream:-/) d++; t++} END {printf "直返し %d / %d = %.1f%%\n", d, t, d/t*100}' $M/access.log
```

- `reqtime ≈ apptime` → プロキシは無罪。時間はアプリとその先にある
- `reqtime ≫ apptime` → プロキシ ↔ クライアント間の転送（レスポンスサイズ）が重い。アプリを速くしても縮まない

次に `slow-digest.txt` の `# Overall` にある **`Exec time` の total** と `apptime` 合計を比べる。

```
DB 実行時間 / apptime 合計 が大半    → DB がボトルネック
そうでない                          → アプリ自身（テンプレート描画、シリアライズ、外部プロセス起動など）
```

### 3. どのエンドポイントか

```bash
head -20 $M/alp.txt
```

`--sort sum` なので**先頭が合計時間を最も食っているエンドポイント**。
`AVG` が小さくても `COUNT` が多ければ上位に来る。ここが実際の改善対象。

### 4. どのクエリか

```bash
sed -n '/^# Overall/,/^# Profile/p' $M/slow-digest.txt | grep -E 'Overall|Exec time|Rows sent|Rows examine'
sed -n '/^# Profile/,/^$/p' $M/slow-digest.txt
```

先頭数本で全体の何 % を占めるかを見る。各クエリの詳細は `# Query N:` ブロックにあり、
`Exec time` 行に **avg / 95% / median** が並ぶ。

> **`Rows examine` と `Rows sent` の比を必ず見る。**
> 走査 646M に対し返却 1.23M なら 525 倍の無駄読みで、インデックス欠如がほぼ確定する。
> 1 回あたりの走査行数がテーブル全体の行数に近ければフルスキャン。

`ADMIN PREPARE` が上位に来たら prepared statement の準備コスト。
Go の MySQL ドライバなら DSN に `interpolateParams=true` で消える。

### 5. キャッシュの効き具合

```bash
# DB の buffer pool ヒット率
join $M/mysql-before.txt $M/mysql-after.txt | awk '
  $1=="Innodb_buffer_pool_read_requests"{rr=$3-$2} $1=="Innodb_buffer_pool_reads"{r=$3-$2}
  END {printf "buffer pool hit %.4f%% (disk reads %d)\n", (1-r/rr)*100, r}'

# クエリ種別ごとのスループット
grep -E 'Com_select|Com_insert|Com_update|Com_delete|Innodb_rows_read|Max_used_connections' $M/mysql-delta.txt

# memcached
join -j2 <(grep ^STAT $M/mc-before.txt|sort -k2) <(grep ^STAT $M/mc-after.txt|sort -k2) \
  | awk '$3 ~ /^[0-9]+$/ && $5 ~ /^[0-9]+$/ && $5-$3!=0 {printf "%-22s %10d\n", $1, $5-$3}' \
  | grep -E 'cmd_get|cmd_set|get_hits|get_misses|evictions'

# プロキシのキャッシュヒット率（proxy_cache 未設定なら全行 cache:- で 0%）
awk -F'\t' '{for(i=1;i<=NF;i++) if($i ~ /^cache:/) c[$i]++} END {for(k in c) print k, c[k]}' $M/access.log
```

buffer pool ヒット率が 99.99% 以上なら**ディスクは効いていない**。
この状態で「BLOB を DB の外に出す」改善をしても効果は薄い。ヒット率が落ちてきたら着手時期。

memcached のヒット率 100% は「よく効いている」とは限らない。**呼び出し回数が少なければ単に使われていないだけ。**
必ず `cmd_get` の実数とスループットを併記する。

## 飽和判定 — 結論を書く前に必ず通す

**「アプリが 100% でない = アプリに余裕がある」とは限らないし、「アプリが 1 位 = アプリがボトルネック」とも限らない。**
どのコンポーネントも 100% に届いていないときは、次の 3 つを確認してから結論を書く。

### (a) cgroup で絞られているか

```bash
for c in app mysql; do
  docker compose -f $COMPOSE exec -T $c cat /sys/fs/cgroup/cpu.stat \
    | grep -E 'nr_periods|nr_throttled|throttled_usec' | sed "s/^/$c /"
done
```

ベンチ前後で差分を取る。`nr_throttled` が `nr_periods` に対して無視できるほど小さければ
（例: 624 周期中 1 回）、**CPU quota では絞られていない**。逆に頻繁なら平均値が低くても実質飽和している。

### (b) そもそも負荷が足りているか

多くのベンチマーカーは**同時実行数が固定**。ソースで上限を確認する
（Go 製なら `makeChanBool` / `errgroup.SetLimit` / ワーカー数の定数）。
各シナリオの中が逐次実行なら、サーバーが同時に見るリクエストはその本数が上限になる。

リトルの法則で実効並列度を出す:

```bash
awk -F'\t' '{for(i=1;i<=NF;i++){split($i,a,":"); v[a[1]]=substr($i,length(a[1])+2)} r+=v["reqtime"]} \
  END {printf "実効並列度 = %.2f\n", r/'"$(cat $M/elapsed.txt)"'}' $M/access.log
```

**実効並列度 = 総 reqtime ÷ 計測秒数。**
これが上限より明らかに小さければ、差分は負荷源自身の処理時間（HTML パース、レスポンス検証）。
つまり**サーバーが遅いのではなく、頼まれていない**。

決定的な確認は、ベンチマーカーを使わず高並列で直接叩くこと:

```bash
docker run --rm --network $NET curlimages/curl:latest sh -c '
  end=$(( $(date +%s) + 25 ))
  i=0; while [ $i -lt 50 ]; do
    ( while [ $(date +%s) -lt $end ]; do curl -s -o /dev/null http://nginx/; done ) &
    i=$((i+1))
  done; wait'
```

これで CPU が上がるなら、ベンチ中の低い数字は**負荷不足**が原因。

### (c) ホスト / VM 自体が天井になっていないか

```bash
docker info --format 'NCPU={{.NCPU}}'
awk -F, 'NR>1 && $3!="" {t[$1]+=$3} END {s=0;n=0; for(k in t){s+=t[k];n++}; printf "全コンテナ合計 %.1f%%\n", s/n}' $M/stats.csv
```

Docker Desktop の VM は既定で CPU が少ない（実測例: ホスト 11 コアに対し VM は 2）。
全コンテナの合計が `NCPU × 100%` に近ければ、**個々のコンテナが 100% に達する前に VM が尽きている**。
負荷源を同じ VM 内で動かしていれば、その分も加算される。

## レポートの構成

次の順で書く。

1. **メタ情報** — 計測日、加えた変更（差分が 1 つなら SQL やコードをそのまま貼る）、`bench.json` の生値
2. **結論** — 1〜2 文。「ボトルネックは X。原因は Y」。ここだけ読めば次の手が決まるように
3. **どのコンポーネントが飽和しているか** — CPU / メモリの表。前回との比較列を入れる
4. **時間配分** — reqtime / apptime / DB 実行時間の内訳。合計秒数と割合を両方
5. **エンドポイント別** — alp の表。前回の AVG を併記して倍率を出す
6. **クエリ別** — pt-query-digest の Profile。**走査行数/回を必ず載せる**
7. **補助指標** — buffer pool ヒット率、キャッシュ、コネクション数
8. **予想と違った点** — 独立した項として書く。ここが次の調査の起点になる
9. **次の一手** — 優先度順。各項目に「なぜその順位か」の数字を添える
10. **踏んだ落とし穴** — 現象 / 原因 / 対処の表。再実行時に同じ穴に落ちないため

### 2 回目以降は「差分」を主役にする

読み手が知りたいのは絶対値ではなく**何がどれだけ変わったか**と**ボトルネックがどこへ移ったか**。

- スコアと主要指標の before → after 表を冒頭に置く
- 狙ったクエリが実際にどうなったかを示す。**走査行数/回の before → after が最も雄弁**
- **新しく上位に来たもの**を明示する。これが次のレポートの主役になる
- 直した項目も表に残し、どれだけ落ちたかを見せる（消すと効果が伝わらない）

### 書くときの注意

- 相対値だけでなく実数も書く（「28.3%」だけでなく「23.5s / 1,120 calls」）
- 比較は倍率まで出す（「速くなった」ではなく「4.291s → 0.154s、÷28」）
- 初期化用エンドポイントは 1 回しか呼ばれない。件数の少ないものを過大評価しない
- 静的ファイルは 1 本あたりは速いが件数が多い。**合計時間で判断する**
- 計測中に踏んだ罠は必ず記録する。同じ環境で必ずまた踏む

## HTML レポートにする場合

`Artifact` で公開する。

- 冒頭に結論とスコアを置く。表とグラフはその後
- 2 回目以降は before/after のペア棒グラフと、積み上げバーの反転（時間配分の逆転）が効く
- グラフの配色は `dataviz` スキルの `validate_palette.js` で検証してから使う。
  グレーに寄った色は「彩度不足」で弾かれるので、検証を通った値に差し替える
- 同じプロジェクトのレポートはタイポグラフィと配色を揃え、一組の資料に見えるようにする
- 回をまたぐときは前回のレポート URL を冒頭からリンクする
