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
Host shanai-isucon-app-01
  IdentityFile ~/.ssh/xxx.pem
  HostName xxx.xxx.xxx.xxx
  User ubuntu
```
