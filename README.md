# 社内ISUCON 改

2016年に作成した社内ISUCONリポジトリを2021年に手直ししました。

［2016年開催時のブログ］

* ISUCON6出題チームが社内ISUCONを開催！AMIも公開！！ - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/05/18/115206
* 社内ISUCONを公開したら広く使われた話 - pixiv inside [archive] https://devpixiv.hatenablog.com/entry/2016/09/26/130112

過去ISUCON公式で練習問題として推奨されたことがある。

* ISUCON初心者のためのISUCON7予選対策 : ISUCON公式Blog https://isucon.net/archives/50697356.html

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

* Ruby/Go/PHPの3言語が用意されており、デフォルトはRubyが起動する
  * Node.jsは現状メンテナンスされていない
  * AMI・Vagrantで他の言語の実装を動かす場合は[manual.md](/manual.md)を参考にする
* AMI・Docker Compose・Vagrantが用意されている
  * 手元で適当に動かすことも難しくない
  * Ansibleを動かせば、他の環境でも動くはず

### AMI

セキュリティのアップデートなどは行わないので自己責任で利用してください。Node.jsのセットアップはskipしているので、Ruby/PHP/Goのみ利用可能。

以下のAMI IDで起動する。リージョンは『Asia Pacific (Tokyo)』。

| 用途           |        AMI ID         |              AMI name               | 推奨インスタンスタイプ |
| -------------- | :-------------------: | :---------------------------------: | ---------------------- |
| 競技者用       | ami-0912fb32e811b9b72 |    catatsuy_private_isu_20220226    | c4.large               |
| ベンチマーカー | ami-0e0d605d5f45cdb72 | catatsuy_private_isu_bench_20220226 | c5.xlarge              |

* `c4.large`などコンピューティング最適化インスタンスで一番小さいインスタンスでの利用を推奨
* 競技者用インスタンスはSecurity groupで80番ポートを公開する必要がある
* ベンチマーカー用インスタンスはコンピューティング最適化インスタンスでそれなりのスペックでの利用を推奨
  * 以下のコマンドでベンチマーカーが実行できる

```sh
$ sudo su - isucon
$ /home/isucon/private_isu.git/benchmarker/bin/benchmarker -u /home/isucon/private_isu.git/benchmarker/userdata -t http://<target IP>
```

最初はRuby実装が起動しているので、他の言語を使用する場合は[manual.md](/manual.md)を見て作業すること。

### 手元で動かす

__いずれの手順もディスク容量が十分にあるマシン上で行うこと__

* アプリケーションは各言語の開発環境とMySQL・memcachedがインストールされていれば動くはず
* ベンチマーカーはGoの開発環境とuserdataがあれば動く
* Dockerとvagrantはメモリが潤沢なマシンで実行すること

#### MacやLinux上で適当に動かす

MySQLとmemcachedを起動した上で以下の手順を実行。

* Ruby実装以外は各言語実装の動かし方を各自調べること
* MySQLのrootユーザーのパスワードが設定されていない前提になっているので、設定されている場合は適宜読み替えること

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

#### Docker Compose

アプリケーションは以下の手順で実行できる。dump.sqlを配置しないとMySQLに初期データがimportされないので注意。

```sh
cd webapp/sql
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/dump.sql.bz2
bunzip2 dump.sql.bz2

cd ..
docker compose up
```

デフォルトはRubyのため、他言語にする場合は`docker-compose.yml`ファイル内のappのbuildを変更する必要がある。PHPはそれに加えて以下の作業が必要。

```sh
cd webapp/etc
mv nginx/conf.d/default.conf nginx/conf.d/default.conf.org
mv nginx/conf.d/php.conf.org nginx/conf.d/php.conf
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

Linuxの場合は`host.docker.internal`が使用できないので、`ip a`してdocker0のインタフェースでホストのIPアドレスを調べて指定する。以下の場合は`172.17.0.1`を指定する。

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

https://github.com/matsuu/cloud-init-isucon/tree/main/private-isu


### 競技者用・ベンチマーカーインスタンスのセットアップ方法

自分で立ち上げたい人向け。`provisioning/`ディレクトリ以下参照。

## 他の言語実装

* Python実装 https://github.com/methane/pixiv-isucon2016-python
* Scala実装 https://github.com/catatsuy/private-isu/pull/140
