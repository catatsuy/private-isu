# private-isu

## ディレクトリ構成

```
├── benchmarker  # ベンチマーカーのソースコード
└── webapp       # 各言語の参考実装
```

* [manual.md](/manual.md)は当日マニュアル。一部社内イベントを意識した記述があるので注意すること。
* [public_manual.md](/public_manual.md) は事前公開レギュレーション


## 手元で動かす

### Docker Compose

アプリケーションは以下の手順で実行できる。dump.sqlを配置しないとMySQLに初期データがimportされないので注意。

```sh
cd webapp/sql
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/dump.sql.bz2
bunzip2 dump.sql.bz2

cd ..
docker compose up
```

ベンチマーカーは以下の手順で実行できる。

```sh
cd benchmarker/userdata
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/img.zip
unzip img.zip
rm img.zip
cd ..

docker build -t private-isu-benchmarker .
docker run --network host -i private-isu-benchmarker /opt/go/bin/benchmarker -t http://host.docker.internal -u /opt/go/userdata
```