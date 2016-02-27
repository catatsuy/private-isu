# private-isu

```
mysql -uroot
CREATE DATABASE isuconp;

cd webapp/sql
mysql -uroot isuconp < schema.sql

cd webapp/scripts
bundle install --path=vendor/bundle
bundle exec ruby create_user.rb

cd webapp/ruby
bundle install --path=vendor/bundle
bundle exec foreman start

cd benchmarker
make
./bin/benchmarker -t "localhost:8080"
```

