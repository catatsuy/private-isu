# private-isu

「[ISUCON](https://isucon.net)」は、LINE株式会社の商標または登録商標です。

本リポジトリが書籍の題材になりました。詳しくは以下のURLをご覧ください。

* [達人が教えるWebパフォーマンスチューニング 〜ISUCONから学ぶ高速化の実践：書籍案内｜技術評論社](https://gihyo.jp/book/2022/978-4-297-12846-3)
* [tatsujin-web-performance/tatsujin-web-performance: 達人が教えるWebパフォーマンスチューニング〜ISUCONから学ぶ高速化の実践](https://github.com/tatsujin-web-performance/tatsujin-web-performance)

ハッシュタグ： `#ISUCON本`

## タイムライン

2016年に作成した社内ISUCONリポジトリを2021年に手直ししました。2022年に書籍の題材になりました。

［2016年開催時のブログ］

* ISUCON6出題チームが社内ISUCONを開催！AMIも公開！！ - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/05/18/115206
* 社内ISUCONを公開したら広く使われた話 - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/09/26/130112

過去ISUCON公式で練習問題として推奨されたことがある。

* ISUCON初心者のためのISUCON7予選対策 : ISUCON公式Blog https://isucon.net/archives/50697356.html

［2021年開催時のブログ］

* 社内ISUCON “TIMES-ISUCON” を開催しました！ | PR TIMES 開発者ブログ https://developers.prtimes.jp/2021/06/04/times-isucon-1/

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

Ubuntu 22.04

## 起動方法

* Ruby/Go/PHPの3言語が用意されており、デフォルトはRubyが起動する
  * Node.jsは現状メンテナンスされていない
  * AMI・Vagrantで他の言語の実装を動かす場合は[manual.md](/manual.md)を参考にする
* AMI・Docker Compose・Vagrantが用意されている
  * 手元で適当に動かすことも難しくない
  * Ansibleを動かせば、他の環境でも動くはず
  * cloud-initも利用可能

### AMI

セキュリティのアップデートなどは行わないので自己責任で利用してください。Node.jsのセットアップはskipしているので、Ruby/PHP/Goのみ利用可能。

* 競技者用インスタンスはSecurity groupで80番ポートを公開する必要がある
  * Network settingsで「Allow HTTP traffic from the internet」にチェックを入れてもよい
* ベンチマーカー用インスタンスはコンピューティング最適化インスタンスでそれなりのスペックでの利用を推奨

ベンチマーカー用インスタンスのベンチマーカー実行方法

```sh
$ sudo su - isucon
$ /home/isucon/private_isu.git/benchmarker/bin/benchmarker -u /home/isucon/private_isu.git/benchmarker/userdata -t http://<target IP>
```

競技者用インスタンス上でのベンチマーカー実行方法（アプリケーションと同居する形になるため非推奨）

```sh
$ sudo su - isucon
$ /home/isucon/private_isu/benchmarker/bin/benchmarker -u /home/isucon/private_isu/benchmarker/userdata -t http://localhost
```

最初はRuby実装が起動しているので、他の言語を使用する場合は[manual.md](/manual.md)を見て作業すること。

以下のAMI IDで起動する。リージョンは『Asia Pacific (Tokyo)』。

競技者用 (Ubuntu 24.04):

| Arch   |                                                                      AMI ID                                                                      | 推奨インスタンスタイプ |
| ------ | :----------------------------------------------------------------------------------------------------------------------------------------------: | ---------------------- |
| x86_64 | [ami-047fdc2b851e73cad](https://ap-northeast-1.console.aws.amazon.com/ec2/home?region=ap-northeast-1#ImageDetails:imageId=ami-047fdc2b851e73cad) | c7a.large              |
| arm64  | [ami-0bed62bba4100a4b7](https://ap-northeast-1.console.aws.amazon.com/ec2/home?region=ap-northeast-1#ImageDetails:imageId=ami-0bed62bba4100a4b7) | c7g.large              |

ベンチマーカー (Ubuntu 24.04):

| Arch   |                                                                      AMI ID                                                                      | 推奨インスタンスタイプ |
| ------ | :----------------------------------------------------------------------------------------------------------------------------------------------: | ---------------------- |
| x86_64 | [ami-037be39355baf1f2e](https://ap-northeast-1.console.aws.amazon.com/ec2/home?region=ap-northeast-1#ImageDetails:imageId=ami-037be39355baf1f2e) | c7a.xlarge             |
| arm64  | [ami-034a457f6af55d65d](https://ap-northeast-1.console.aws.amazon.com/ec2/home?region=ap-northeast-1#ImageDetails:imageId=ami-034a457f6af55d65d) | c7g.xlarge             |

### 手元で動かす

__いずれの手順もディスク容量が十分にあるマシン上で行うこと__

* アプリケーションは各言語の開発環境とMySQL・memcachedがインストールされていれば動くはず
* ベンチマーカーはGoの開発環境とuserdataがあれば動く
* Dockerとvagrantはメモリが潤沢なマシンで実行すること

#### 初期データを用意する

必要になるので、以下の手順を行う前に必ず実行すること。

```sh
make init
```

#### MacやLinux上で適当に動かす

MySQLとmemcachedを起動した上で以下の手順を実行。

* Ruby実装以外は各言語実装の動かし方を各自調べること
* MySQLのrootユーザーのパスワードが設定されていない前提になっているので、設定されている場合は適宜読み替えること

```sh
bunzip2 -c webapp/sql/dump.sql.bz2 | mysql -uroot

cd webapp/ruby
bundle install --path=vendor/bundle
bundle exec foreman start
cd ../..

cd benchmarker
make
./bin/benchmarker -t "http://localhost:8080" -u ./userdata
# ./bin/benchmarker -t "http://<競技者用インスタンスのグローバルIPアドレス>/" -u ./userdata

# Output
# {"pass":true,"score":1710,"success":1434,"fail":0,"messages":[]}
```

#### Docker Compose

アプリケーションは以下の手順で実行できる。dump.sql.bz2を配置しないとMySQLに初期データがimportされないので注意。

```sh
cd webapp
docker compose up
```

（もしうまく動かなければ`docker-compose up`を使うとよいかもしれません）

デフォルトはRubyのため、他言語にする場合は`docker-compose.yml`ファイル内のappのbuildを変更する必要がある。PHPはそれに加えて以下の作業が必要。

```sh
cd webapp/etc
mv nginx/conf.d/default.conf nginx/conf.d/default.conf.org
mv nginx/conf.d/php.conf.org nginx/conf.d/php.conf
```

ベンチマーカーは以下の手順で実行できる。

```sh
cd benchmarker
docker build -t private-isu-benchmarker .
docker run --network host -i private-isu-benchmarker /bin/benchmarker -t http://host.docker.internal -u /opt/userdata
# Linuxの場合
docker run --network host --add-host host.docker.internal:host-gateway -i private-isu-benchmarker /bin/benchmarker -t http://host.docker.internal -u /opt/userdata
```

動かない場合は`ip a`してdocker0のインタフェースでホストのIPアドレスを調べて`host.docker.internal`の代わりに指定する。以下の場合は`172.17.0.1`を指定する。

```
3: docker0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue state DOWN group default
    link/ether 02:42:ca:63:0c:59 brd ff:ff:ff:ff:ff:ff
    inet 172.17.0.1/16 brd 172.17.255.255 scope global docker0
       valid_lft forever preferred_lft forever
    inet6 fe80::42:caff:fe63:c59/64 scope link
       valid_lft forever preferred_lft forever
```

#### Vagrant

手元にansibleをインストールして`vagrant up`すればprovisioningが実行される。

benchからappのIPアドレスを指定して負荷をかける。

```shell
# appのIPアドレスを調べる
$ vagrant ssh app
$ ip a

3: enp0s8: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP group default qlen 1000
    link/ether 08:00:27:37:2b:2c brd ff:ff:ff:ff:ff:ff
    inet 172.28.128.6/24 brd 172.28.128.255 scope global dynamic enp0s8
       valid_lft 444sec preferred_lft 444sec
    inet6 fe80::a00:27ff:fe37:2b2c/64 scope link
       valid_lft forever preferred_lft forever

# benchで負荷をかける
$ vagrant ssh bench
$ sudo su - isucon
$ /home/isucon/private_isu.git/benchmarker/bin/benchmarker -u /home/isucon/private_isu.git/benchmarker/userdata -t http://172.28.128.6
```

最初はRuby実装が起動しているので、他の言語を使用する場合は[manual.md](/manual.md)を見て作業すること。

#### cloud-init を利用して環境を構築する

matsuuさんの[cloud-initに対応した環境でISUCONの過去問を構築するためのcloud-config集](https://github.com/matsuu/cloud-init-isucon/)を利用して競技者用・ベンチマーカーインスタンスの構築ができます。

cloud-initに対応した環境、例えばAWS、Azure、Google Cloud、Oracle Cloud、さくらのクラウド、Multipass、VMwareなど、クラウドからローカルまで幅広く環境構築が可能です。

Apple Silicon搭載のマシン上で動作させる場合はMultipassを利用することを推奨します。

https://github.com/matsuu/cloud-init-isucon/tree/main/private-isu

ISUCON過去問題の環境を「さくらのクラウド」で構築する | さくらのナレッジ https://knowledge.sakura.ad.jp/31520/

#### Cloud Formationを利用して構築する

https://gist.github.com/tohutohu/024551682a9004da286b0abd6366fa55 を参照

### 競技者用・ベンチマーカーインスタンスのセットアップ方法

自分で立ち上げたい人向け。`provisioning/`ディレクトリ以下参照。

### 事例集

* private-isuのベンチマーカーをLambdaで実行する仕組みを公開しました | PR TIMES 開発者ブログ https://developers.prtimes.jp/2024/01/29/private-isu-bench-lambda/
* 日本CTO協会による合同ISUCON研修の紹介 - Pepabo Tech Portal https://tech.pepabo.com/2024/02/16/isucon-2023/

## 他の言語実装

* Python実装 https://github.com/methane/pixiv-isucon2016-python
  * provisioning code https://github.com/x-tech5/aws-isucon-book-tutorial
* Rust実装 https://github.com/Romira915/private-isu-rust
* Scala実装 https://github.com/catatsuy/private-isu/pull/140
