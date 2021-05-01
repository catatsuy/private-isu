# 社内ISUCON 改

2016年に作成した社内ISUCONリポジトリを2021年に手直ししました。

  * ISUCON6出題チームが社内ISUCONを開催！AMIも公開！！ - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/05/18/115206
  * 社内ISUCONを公開したら広く使われた話 - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/09/26/130112

## ディレクトリ構成

```
├── ansible      # ベンチマーカー用ansible（非推奨）
├── benchmarker  # ベンチマーカーなどが依存するパッケージのソースコード
├── provisioning # 競技者用インスタンスセットアップ用ansible
└── webapp       # 各言語の参考実装
```

* [manual.md](/manual.md)は当日マニュアル。一部社内イベントを意識した記述があるので注意すること。
* [public_manual.md](/public_manual.md) は事前公開レギュレーション

## 起動方法

### ベンチマーカー

以下の手順で実行できる。

```sh
/opt/go/bin/benchmarker -t http://<競技者用インスタンスのグローバルIPアドレス>/ -u /opt/go/src/github.com/catatsuy/private-isu/benchmarker/userdata

# Output
# {"pass":true,"score":1710,"success":1434,"fail":0,"messages":[]}
```

## 競技用インスタンスのセットアップ方法

自分で立ち上げたい人向け。`provisioning/`ディレクトリ以下参照。

## 適当に手元で立てる

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

## 他の言語実装

* Python実装 https://github.com/methane/pixiv-isucon2016-python
* Scala実装 https://github.com/catatsuy/private-isu/pull/140
