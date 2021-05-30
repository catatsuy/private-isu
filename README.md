# 社内ISUCON 改

2016年に作成した社内ISUCONリポジトリを2021年に手直ししました。

  * ISUCON6出題チームが社内ISUCONを開催！AMIも公開！！ - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/05/18/115206
  * 社内ISUCONを公開したら広く使われた話 - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/09/26/130112

「[ISUCON](https://isucon.net)」は、LINE株式会社の商標または登録商標です。

## ディレクトリ構成

```
├── ansible_old  # ベンチマーカー・portal用ansible（非推奨）
├── benchmarker  # ベンチマーカーのソースコード
├── portal       # portal（非推奨）
├── provisioning # 競技者用インスタンスセットアップ用ansible
└── webapp       # 各言語の参考実装
```

* [manual.md](/manual.md)は当日マニュアル。一部社内イベントを意識した記述があるので注意すること。
* [public_manual.md](/public_manual.md) は事前公開レギュレーション

## OS

Ubuntu 20.04

## 起動方法

### AMI

競技者用インスタンスのみ用意。2021/05/29に作成。セキュリティのアップデートなどは行わないので自己責任で利用してください。
Node.jsのセットアップはskipしているので、Ruby/PHP/Goのみ利用可能。

```
AMI ID: ami-01d454481527ddc3d
AMI Name: catatsuy_private_isu_20210530
```

メモリが1GBに制限されるため、`c4.large`などコンピューティング最適化インスタンスで一番小さいインスタンスでの利用を推奨。

### 適当に手元で試す

ベンチマーカーはGoとuserdataがあれば動かせる。以下の手順で実行できる。

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
./bin/benchmarker -t "http://localhost:8080" -u $PWD/userdata
# ./bin/benchmarker -t "http://<競技者用インスタンスのグローバルIPアドレス>/" -u $PWD/userdata

# Output
# {"pass":true,"score":1710,"success":1434,"fail":0,"messages":[]}
```

### 競技用インスタンスのセットアップ方法

自分で立ち上げたい人向け。`provisioning/`ディレクトリ以下参照。

## 他の言語実装

* Python実装 https://github.com/methane/pixiv-isucon2016-python
* Scala実装 https://github.com/catatsuy/private-isu/pull/140
