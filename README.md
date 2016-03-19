# private-isu

```
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
./bin/benchmarker -t "localhost:8080" -u $PWD/userdata
```

