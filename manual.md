# 社内ISUCON 当日マニュアル

## 当日の流れ

  * 12:30 競技開始
  * 19:00 競技終了

## ポータルサイト

上記リンクを開いてください。計測ツールで測定したスコアはこのポータルに送られ、集計結果を見ることができます。

## Getting Started

はじめに以下の操作を行い、問題なく動くかを確認して下さい。

### 2. 起動したEC2インスタンスに `ubuntu` ユーザで SSH ログインする

例:

```
ssh -i <設定した鍵ファイル> ubuntu@xx.xx.xx.xx
```

ログイン後に`isucon`ユーザーでログインできるようにすることをおすすめします。

### 3. アプリケーションの動作を確認

EC2インスタンスのパブリックIPアドレスにブラウザでアクセスし、動作を確認してください。以下の画面が表示されるはずです。

例として、「アカウント名」は `mary`、 「パスワード」は `marymary` を入力することでログインが行えます。

ブラウザでアクセスできない場合、主催者に確認してください。

### 4. 負荷走行を実行

この操作後、ポータルにて、あなたのチームのスコアが反映されているか確認して下さい。負荷走行を実行すると、あなたのアプリケーションに対して自動的にリクエストが送信され、その結果がポータルサイトのスコアに反映されます。スコアの反映には数分かかる場合があります。

### ディレクトリ構成

参考実装のアプリケーションコードおよび、スコア計測用プログラムは `/home/isucon` ディレクトリ以下にあります。

```
/home/isucon/
  ├ env.sh       # アプリケーション用の環境変数
  └ private_isu/
     ├ webapp/    # 各言語の参考実装
     ├ manual.md    # 本マニュアル
     └ public_manual.md # 当日レギュレーション
```

### 参考実装の言語切り替え方法

初期状態ではRubyによる参考実装が起動しています。これをベースに最適化を進めるか、必要に応じてPHP、Go、Node.js、またはPythonの参考実装に切り替えることができます。一度に起動できるアプリケーション言語は1つだけです。基本的な切り替え手順は、現在動作しているRubyのサービス(`isu-ruby`)を停止・無効化し、その後、目的の言語のサービスを起動・有効化します。PHPへ切り替える場合、またはPHPからRubyへ戻す場合は、Nginxの設定変更も伴います。

80番ポートでアクセスできるので、ブラウザから動作確認をすることができます。

プログラムの詳しい起動方法は、 /etc/systemd/system/isu-ruby.service を参照してください。

エラーなどの出力については、

```bash
$ sudo journalctl -f -u isu-ruby
```

などで見ることができます。

また、unicornの再起動は、

```bash
$ sudo systemctl restart isu-ruby
```

などですることができます。

#### PHP (php8.3-fpm) への切り替え方

Ruby実装からPHP実装に切り替えるには、以下の操作を行います。まず、Rubyサービスを停止・無効化します:

```bash
$ sudo systemctl stop isu-ruby
$ sudo systemctl disable isu-ruby
```

```bash
$ sudo rm /etc/nginx/sites-enabled/isucon.conf
$ sudo ln -s /etc/nginx/sites-available/isucon-php.conf /etc/nginx/sites-enabled/
$ sudo systemctl reload nginx
```

```bash
$ sudo systemctl start php8.3-fpm
$ sudo systemctl enable php8.3-fpm
```

php-fpmの設定については、`/etc/php/8.3/fpm/` 以下にあります。

エラーなどの出力については、

```bash
$ sudo journalctl -f -u php8.3-fpm
$ sudo tail -f /var/log/nginx/error.log
```

などで見ることができます。

#### Go (isu-go) への切り替え方

Ruby実装からGo実装に切り替えるには、以下の操作を行います。まず、Rubyサービスを停止・無効化します:

```bash
$ sudo systemctl stop isu-ruby
$ sudo systemctl disable isu-ruby
```

```bash
$ sudo systemctl start isu-go
$ sudo systemctl enable isu-go
```

プログラムの詳しい起動方法は、 /etc/systemd/system/isu-go.service を参照してください。

エラーなどの出力については、

```bash
$ sudo journalctl -f -u isu-go
```

などで見ることができます。

#### Python (isu-python) への切り替え方

Ruby実装からPython実装に切り替えるには、以下の操作を行います。まず、Rubyサービスを停止・無効化します:

```bash
$ sudo systemctl stop isu-ruby
$ sudo systemctl disable isu-ruby
```

```bash
# Python 用 systemd ユニットを有効化・起動
$ sudo systemctl start isu-python
$ sudo systemctl enable isu-python
```

プログラムの詳しい起動方法は、`/etc/systemd/system/isu-python.service`を参照してください。

```bash
# リアルタイムでログを追う
$ sudo journalctl -f -u isu-python
```

#### Node.js (isu-node) への切り替え方

Ruby実装からNode.js実装に切り替えるには、以下の操作を行います。まず、Rubyサービスを停止・無効化します:

```bash
$ sudo systemctl stop isu-ruby
$ sudo systemctl disable isu-ruby
```

```bash
$ sudo systemctl start isu-node
$ sudo systemctl enable isu-node
```

プログラムの詳しい起動方法は、 /etc/systemd/system/isu-node.service を参照してください。

```bash
$ sudo journalctl -f -u isu-node
```

### MySQL

3306番ポートでMySQLが起動しています。初期状態では以下のユーザが設定されています。

  * ユーザ名: `isuconp`, パスワード: `isuconp`

### memcached

11211番ポートでmemcachedが起動しています。


## ルール詳細

[社内ISUCON 当日レギュレーション](/public_manual.md)

本マニュアルは、競技環境の技術的な詳細と操作手順を提供します。当日レギュレーション (`public_manual.md`) には競技全体のルールが記載されています。原則として、競技ルールについては `public_manual.md` を、技術的な操作や環境については本マニュアルを参照してください。

### スコアについて

基本スコアは以下のルールで算出されます。

```
成功レスポンス数(GET) x 1 + 成功レスポンス数(POST) x 2 + 成功レスポンス数(画像投稿) x 5 - (サーバエラー(error)レスポンス数 x 10 + リクエスト失敗(exception)数 x 20 + 遅延POSTレスポンス数 x 100)
```

ただし、基本スコアと計測ツールの出すスコアが異なっている場合は、計測ツールの出すスコアが優先されます。

#### 減点対象

以下の事項に抵触すると減点対象となります。

  * 存在するべきファイルへのアクセスが失敗する
  * リクエスト失敗（通信エラー等）が発生する
  * サーバエラー(Status 5xx)・クライアントエラー(Status 4xx)をアプリケーションが返す
  * 他、計測ツールのチェッカが検出したケース

#### 注意事項

  * リダイレクトはリダイレクト先が正しいレスポンスを返せた場合に、1回レスポンスが成功したと判断します
  * POSTの失敗は大幅な減点対象です

### 制約事項

以下の事項に抵触すると点数が無効となります。

  * GET /initialize へのレスポンスが10秒以内に終わらない
  * 存在するべきDOM要素がレスポンスHTMLに存在しない

## 当日サポートについて

競技中、別途アナウンスされるSlackのチャットルームにて、サポートを行います。

ルールについてや、基本的なトラブルの質問にはお答えできますが、計測ツールおよびアプリケーションに関する質問には原則として回答しません。予めご了承ください。
