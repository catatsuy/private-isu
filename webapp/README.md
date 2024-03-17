# reference

https://note.com/pharmax/n/n27d5ffb576c7

## how to build

環境構築
今回はみんなで触ってみることが目的なので、EC2ではなく気軽に触れるdockerで環境構築しました。
手順を紹介します。

リポジトリのクローンと初期データ投入
```
git clone git@github.com:catatsuy/private-isu.git

cd private-isu/
```

# 初期データの配置
```
cd webapp/sql
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/dump.sql.bz2
bunzip2 dump.sql.bz2
```

# dockerの起動
```
cd ..
docker-compose up
```

ベンチマーカーのビルド
```
cd /benchmarker/userdata
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/img.zip
unzip img.zip
rm img.zip
cd ..

docker build -t private-isu-benchmarker .

#　ベンチマーカー実行
docker run --network host -i private-isu-benchmarker /opt/go/bin/benchmarker -t http://host.docker.internal -u /opt/go/userdata

```


nginxとmysqlのログをホストのlogsディレクトリに出力するようにする
docker-compose.ymlを編集し、logsディレクトリをマウントする。

nginxのvolumeに以下を追加

```
- ./logs/nginx:/var/log/nginx
```


mysqlのvolumeに以下を追加

```
- ./logs/mysql:/var/log/mysql
```


mysqlのスロークエリログを有効にする
/webapp/etc/my.cnfを以下に置き換え

```
[mysqld]
default_authentication_plugin=mysql_native_password
slow_query_log=1
slow_query_log_file=/var/log/mysql/slow-query.log
long_query_time=0.0
```


この辺の設定の意味などは、実際に解くタイミングで説明しようと思います。

nginxのログ出力設定
/webapp/etc/nginx/conf.d/default.confを以下に置き換え
後でalpを使って分析するため、ltsv形式でログを保存するようにする。

```
log_format ltsv "time:$time_local"
  "\thost:$remote_addr"
  "\tforwardedfor:$http_x_forwarded_for"
  "\treq:$request"
  "\tmethod:$request_method"
  "\turi:$request_uri"
  "\tstatus:$status"
  "\tsize:$body_bytes_sent"
  "\treferer:$http_referer"
  "\tua:$http_user_agent"
  "\treqtime:$request_time"
  "\truntime:$upstream_http_x_runtime"
  "\tapptime:$upstream_response_time"
  "\tcache:$upstream_http_x_cache"
  "\tvhost:$host";
  
server {
  listen 80;
  client_max_body_size 10m;
  root /public/;

  location / {
    proxy_set_header Host $host;
    proxy_pass http://app:8080;
  }

  access_log  /var/log/nginx/access.log ltsv;
}
```



dockerを再起動し、再度ベンチマーカー実行
ベンチマーカーを再度実行し、ログを出力させる

```
docker-compose up

docker run --network host -i private-isu-benchmarker /opt/go/bin/benchmarker -t http://host.docker.internal -u /opt/go/userdata

```

nginxのログ分析ツールの導入
alpをmac側に導入。

```
brew install alp

# ログ確認
cat logs/nginx/access.log | alp ltsv -m="^/posts/[0-9]+","^/image/[0-9]+\.(jpg|png|gif)","^/@[a-z]*" --reverse

```


mysqlのログ分析ツールの導入
pt-query-digestもmacに導入。

```
brew install percona-toolkit

# ログ確認
sudo pt-query-digest logs/mysql/slow-query.log

```


