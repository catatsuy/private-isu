# Pixiv 社内 ISUCON 2016, Python 実装

[Pixiv 社内 ISUCON 2016](https://github.com/catatsuy/private-isu) の Python 版実装です。

Python で ISUCON に参加する予定の人が、 Pixiv 社内 ISUCON を使って練習するときの
参照実装として使えるようにと考えています。

ISUCON本を題材にした練習にも利用できます。

## セットアップ

### チェックアウト

AMI からのスタートでも、 リポジトリからのスタートでも、 `webapp` ディレクトリがあるはずなので、
そこにこのリポジトリをチェックアウトしてください。

```console
$ cd private_isu/webapp
$ git clone https://github.com/methane/pixiv-isucon2016-python python
$ cd python
```

## 実行環境の準備の準備

private-isuのAMI(Ubuntu 22.04 LTS)の場合は以下を実行しておきます。

```console
$ sudo apt update
$ sudo apt -y install python3.10-venv
$ sudo apt -y install python3.10-dev default-libmysqlclient-dev build-essential
```

### 実行環境の準備

pyenv 等を利用して Python 3.10.6 を準備しておいてください。
private-isuのAMI(Ubuntu 22.04 LTS)の場合はシステムの Python 3.10 を利用できます。
以下の例では venv を用意していますが、 pyenv を使ってる場合は venv を利用せず直接インストールしても大丈夫です。

```console
$ python3 -V
Python 3.10.6
$ python3 -m venv venv
$ venv/bin/pip install -r requirements.freeze
```

## デバッグモードで実行

```console
$ FLASK_APP=app.py venv/bin/flask run -p 8080 --debugger
```


## デーモンとして設定

```console
$ venv/bin/pip install gunicorn==20.1.0
$ sudo cp isu-python.service /etc/systemd/system/.
$ sudo systemctl daemon-reload
$ sudo systemctl disable isu-ruby
$ sudo systemctl stop isu-ruby
$ sudo systemctl start isu-python
$ sudo systemctl enable isu-python
```
