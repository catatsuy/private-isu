# 社内ISUCON 改

2016年に作成した社内ISUCONリポジトリを2021年に手直ししました。

［2016年開催時のブログ］

* ISUCON6出題チームが社内ISUCONを開催！AMIも公開！！ - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/05/18/115206
* 社内ISUCONを公開したら広く使われた話 - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/09/26/130112

［2021年開催時のブログ］

* 社内ISUCON “TIMES-ISUCON” を開催しました！ | PR TIMES 開発者ブログ https://developers.prtimes.jp/2021/06/04/times-isucon-1/

「[ISUCON](https://isucon.net)」は、LINE株式会社の商標または登録商標です。

## ディレクトリ構成

```
├── ansible_old  # ベンチマーカー・portal用ansible（非推奨）
├── benchmarker  # ベンチマーカーのソースコード
├── portal       # portal（非推奨）
├── provisioning # 競技者用・ベンチマーカーインスタンスセットアップ用ansible
└── webapp       # 各言語の参考実装
```

* [manual.md](/manual.md)は当日マニュアル。一部社内イベントを意識した記述があるので注意すること。
* [public_manual.md](/public_manual.md) は事前公開レギュレーション

## OS

Ubuntu 20.04

## 起動方法

### AMI

セキュリティのアップデートなどは行わないので自己責任で利用してください。Node.jsのセットアップはskipしているので、Ruby/PHP/Goのみ利用可能。

以下のAMI IDで起動する。リージョンは『Asia Pacific (Tokyo)』。

| 用途           |        AMI ID         | 推奨インスタンスタイプ |
| -------------- | :-------------------: | ---------------------- |
| 競技者用       | ami-08f7463a003fd68a3 | c4.large               |
| ベンチマーカー | ami-04a7c6eb0387c8bc8 | c5.xlarge              |

* 競技者用インスタンスはメモリが1GBに制限されるため、`c4.large`などコンピューティング最適化インスタンスで一番小さいインスタンスでの利用を推奨
  * t系インスタンスだと不安定になるみたいなので注意
* ベンチマーカー用インスタンスはコンピューティング最適化インスタンスでそれなりのスペックでの利用を推奨
  * 以下のコマンドでベンチマーカーが実行できる

```sh
/home/isucon/private_isu.git/benchmarker/bin/benchmarker -u /home/isucon/private_isu.git/benchmarker/userdata -t http://<target IP>
```

### 適当に手元で試す

* アプリケーションは各言語の開発環境とMySQL・memcachedがインストールされていれば動くはず
* ベンチマーカーはGoの開発環境とuserdataがあれば動く

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

他にもVagrantや一部の言語はDocker Composeも用意している

### Docker Compose

```sh
cd webapp/sql
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/dump.sql.bz2
bunzip2 dump.sql.bz2

cd ..
docker-compose up
```

デフォルトはRubyのため、他言語にする場合は`docker-compose.yml`ファイル内のappのbuildを変更する必要がある。PHPはそれに加えて以下の作業が必要。

```sh
cd webapp/etc
mv nginx/conf.d/default.conf nginx/conf.d/default.conf.org
mv nginx/conf.d/php.conf.org nginx/conf.d/php.conf
```

### 競技用インスタンスのセットアップ方法

自分で立ち上げたい人向け。`provisioning/`ディレクトリ以下参照。

## 他の言語実装

* Python実装 https://github.com/methane/pixiv-isucon2016-python
* Scala実装 https://github.com/catatsuy/private-isu/pull/140
