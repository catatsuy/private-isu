# 社内ISUCON 2016

## 起動方法

### AMI

以下のAMI IDとインスタンスタイプで起動する。リージョンは『Asia Pacific (Tokyo)』。

| 用途 | AMI ID | インスタンスタイプ |
|---|:---:|---|
| 競技者用 | ami-18b05179 | c4.large |
|  ベンチマーカー | ami-53ef0e32 | c4.xlarge |

### 競技者用インスタンス

[manual.md](/manual.md)を参照のこと（一部社内イベントを意識した記述がある）。

`webapp/`ディレクトリ以下に全言語の実装がある。

### ベンチマーカー用インスタンス

以下の手順で実行できる。

```sh
/opt/go/bin/benchmarker -t http://<競技者用インスタンスのグローバルIPアドレス>/ -u /opt/go/src/github.com/catatsuy/private-isu/benchmarker/userdata

# Output
# {"pass":true,"score":1710,"success":1434,"fail":0,"messages":[]}
```

## 競技用インスタンスのセットアップ方法

自分で立ち上げたい人向け。`provisioning/`ディレクトリ以下参照。

## 適当に手元で立てる

ベンチマーカーのビルドにはリポジトリ自体が`GOPATH`内にある必要がある。

```sh
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/dump.sql.bz2
bzcat dump.sql.bz2 | mysql -uroot

cd webapp/ruby
bundle install --path=vendor/bundle
bundle exec foreman start
cd ../..

cd benchmarker/userdata
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/img.zip
unzip img.zip
cd ../..

cd benchmarker
make
./bin/benchmarker -t "localhost:8080" -u $PWD/userdata
```
