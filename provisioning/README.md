# 競技用インスタンスのセットアップ方法

## webapp

image/ansible以下に入っているplaybookを順番に実行。

```
$ ansible-playbook -i hosts image/ansible/playbooks.yml --skip-tags nodejs
```

## bench

bench/ansible以下に入っているplaybookを順番に実行。

```
$ ansible-playbook -i hosts bench/ansible/playbooks.yml
```

## ssh config の例

```
Host isu-app
  IdentityFile ~/.ssh/xxx.pem
  HostName xxx.xxx.xxx.xxx
  User ubuntu

Host isu-bench
  IdentityFile ~/.ssh/xxx.pem
  HostName xxx.xxx.xxx.xxx
  User ubuntu
```
