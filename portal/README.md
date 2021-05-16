# isucon portal

### require

* Redis
* Sidekiq

```
apt install -y libmysqlclient-dev
apt install -y sqlite3
apt install -y redis
```

## Development

### Portal webapp

```
bundle install -j4 --path=vendor/bundle
bundle exec rake db:setup db:seed
```

```
bundle exec rails s -b 0.0.0.0 -p 3000
```

And open `http://localhost:3000`

## Run in production

### asset:precompile

```
bundle exec rails assets:precompile
```

### Sidekiq

```
bundle exec sidekiq
```

FIXME:
