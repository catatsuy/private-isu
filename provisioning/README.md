# 競技用インスタンスのセットアップ方法

image/ansible以下に入っているplaybookを順番に実行。

```
$ ansible-playbook -i hosts image/ansible/*.yml
```

## ssh config の例

```
Host shanai-isucon-app-01
  IdentityFile ~/.ssh/xxx.pem
  HostName xxx.xxx.xxx.xxx
  User admin
```
