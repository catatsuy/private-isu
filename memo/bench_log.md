# ベンチログ

## 初期状態

```json
{
  "pass": true,
  "score": 315,
  "success": 365,
  "fail": 6,
  "messages": ["リクエストがタイムアウトしました (POST /login)"]
}
```

## nginx のデバッグログを追加

```json
{
  "pass": true,
  "score": 159,
  "success": 353,
  "fail": 13,
  "messages": [
    "リクエストがタイムアウトしました (GET /favicon.ico)",
    "リクエストがタイムアウトしました (POST /login)",
    "リクエストがタイムアウトしました (POST /register)"
  ]
}
```
