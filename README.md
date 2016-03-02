# private-isu

```
mysql -uroot
CREATE DATABASE isuconp;

cd webapp/sql
mysql -uroot isuconp < schema.sql
cd ../..

curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/dump.sql.bz2
bunzip2 dump.sql.bz2
mysql -uroot isuconp < dump.sql

cd webapp/scripts
bundle install --path=vendor/bundle
bundle exec ruby create_user.rb
cd ../..

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

