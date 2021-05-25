# 社内ISUCON 当日マニュアル

## 当日の流れ

 * 10:30 開始
 * 17:30 終了・結果発表
 * 18:00 問題解説・質問タイム
 * 19:00 終了

## ポータルサイト
https://mackerel.io/orgs/PRTIMES/services/TIMES-ISUCON?metrics=service#period=30m 

上記リンクを開いてください。計測ツールで測定したスコアはこのポータルに送られ、集計結果を見ることができます。

## Getting Started

はじめに以下の操作を行い、問題なく動くかを確認して下さい。

### 1. 主催者からSSH秘密鍵を受け取り、`~/.ssh/times-isucon.pem` に配置する
```
vi ~/.ssh/times-isucon.pem
chmod 600 ~/.ssh/times-isucon.pem
```

### 2. EC2インスタンスに `isucon` ユーザで SSH ログインする

例:

```
ssh -i ~/.ssh/times-isucon.pem isucon@xx.xx.xx.xx
```

### 3. アプリケーションの動作を確認

EC2インスタンスのパブリックIPアドレスにブラウザでアクセスし、動作を確認してください。

例として、「アカウント名」は `mary`、 「パスワード」は `marymary` を入力することでログインが行えます。

ブラウザでアクセスできない場合、主催者に確認してください。

### 4. ベンチマーカーを実行

EC2インスタンスにSSHログインして以下のコマンドを実行し、ベンチマーカーを実行してください。
(`APIキー`の部分は主催者から受け取ったAPIキーで置き換えてください。)
```
curl -X POST https://f2cx3ti5rh.execute-api.ap-northeast-1.amazonaws.com/async --header 'x-api-key:APIキー'
```

ベンチマーカーは非同期で実行され、約1分でSlackの`#pj_times-isucon_2021_score`チャンネルに結果が通知されます。

ベンチマーカーの実行が完了したら、ポータルサイトにスコアが反映されていることを確認してください。 

https://mackerel.io/orgs/PRTIMES/services/TIMES-ISUCON?metrics=service#period=30m

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

参考実装の言語はRuby, PHP, Goが用意されており、初期状態ではRubyの実装が起動しています。

80番ポートでアクセスできるので、ブラウザから動作確認をすることができます。

プログラムの詳しい起動方法は、 /etc/systemd/system/isu-ruby.service を参照してください。

エラーなどの出力については、

```
$ sudo journalctl -f -u isu-ruby
```

などで見ることができます。

また、unicornの再起動は、

```
$ sudo systemctl restart isu-ruby
```

などですることができます。

#### PHPへの切り替え方

起動する実装をPHPに切り替えるには、以下の操作を行います。

```
$ sudo systemctl stop isu-ruby
$ sudo rm /etc/nginx/sites-enabled/isucon.conf
$ sudo ln -s /etc/nginx/sites-available/isucon-php.conf /etc/nginx/sites-enabled/
$ sudo systemctl reload nginx
$ sudo systemctl start php7.4-fpm
```

php-fpmの設定については、/etc/php/7.4/fpm/以下にあります。

#### Goへの切り替え方

起動する実装をGOに切り替えるには、以下の操作を行います。

```
$ sudo systemctl stop isu-ruby
$ sudo systemctl start isu-go
```

プログラムの詳しい起動方法は、 /etc/systemd/system/isu-go.service を参照してください。

エラーなどの出力については、

```
$ sudo journalctl -f -u isu-go
```

などで見ることができます。

### MySQL

3306番ポートでMySQLが起動しています。初期状態では以下のユーザが設定されています。

  * ユーザ名: `isuconp`, パスワード: `isuconp`

### memcached

11211番ポートでmemcachedが起動しています。


## ルール詳細

https://gist.github.com/catatsuy/d607f514abbf125e1599ba722ce6942b

なお、当日レギュレーションと本マニュアルの記述に矛盾がある場合、本マニュアルの記述が優先されます。

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
