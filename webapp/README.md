# private-isu-golang

## description
"private-isu-golang" is an ISUCON practice repository using golang.

## create template to use golang

Clone repository
```
git clone git@github.com:catatsuy/private-isu.git
```

Edit yml for using golang
```
# L15
build: golang/

# L43, L44: mysql:volumes:
- ./logs/nginx:/var/log/nginx
- ./logs/mysql:/var/log/mysql
```

Init data setting
```
cd webapp/sql
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/dump.sql.bz2
bunzip2 dump.sql.bz2
```

Docker start
```
cd ..
docker-compose up
```

---

## benchmarker build

```
cd /benchmarker/userdata
curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/img.zip
unzip img.zip
rm img.zip
```

```
cd ..

docker build -t private-isu-benchmarker .

#　execute benchmarker
docker run --network host -i private-isu-benchmarker /opt/go/bin/benchmarker -t http://host.docker.internal -u /opt/go/userdata
```

---

## mysql slowquery log 

```cnf:/webapp/etc/my.cnf
# add my.cnf
[mysqld]
default_authentication_plugin=mysql_native_password
slow_query_log=1
slow_query_log_file=/var/log/mysql/slow-query.log
long_query_time=0.0
```

## nginx log 

```cnf: /webapp/etc/nginx/conf.d/default.conf
log_format ltsv "time:$time_local"
  "\thost:$remote_addr"
  "\tforwardedfor:$http_x_forwarded_for"
  "\treq:$request"
  "\tmethod:$request_method"
  "\turi:$request_uri"
  "\tstatus:$status"
  "\tsize:$body_bytes_sent"
  "\treferer:$http_referer"
  "\tua:$http_user_agent"
  "\treqtime:$request_time"
  "\truntime:$upstream_http_x_runtime"
  "\tapptime:$upstream_response_time"
  "\tcache:$upstream_http_x_cache"
  "\tvhost:$host";
  
server {
  listen 80;
  client_max_body_size 10m;
  root /public/;

  location / {
    proxy_set_header Host $host;
    proxy_pass http://app:8080;
  }

  access_log  /var/log/nginx/access.log ltsv;
}
```

