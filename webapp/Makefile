OS=$(shell uname -s)
ARCH=$(shell uname -m)

prepare:
	docker-compose exec mysql /bin/bash -c "echo -n > /var/log/mysql/mysql-slow.log"
	docker-compose exec nginx /bin/bash -c "echo -n > /var/log/nginx/_access.log"

analyze:
	make slowquery & make accesslog

slowquery: log bin/pt-query-digest
	docker compose cp mysql:/var/log/mysql/mysql-slow.log log/slow.log
	./bin/pt-query-digest log/slow.log > log/analyzed-slow.log
	cat log/analyzed-slow.log

accesslog: log bin/alp
	docker compose cp nginx:/var/log/nginx/_access.log log/access.log
	./bin/alp json \
	  --sort sum -r \
	  -m "/posts/[0-9]+,/@\w+,/image/\d+" \
	  -o count,method,uri,min,avg,max,sum \
	  < log/access.log > log/analyzed-access.log
	cat log/analyzed-access.log

bin/alp: bin
	wget -N -P bin https://github.com/tkuchiki/alp/releases/download/v1.0.21/alp_$(OS)_$(ARCH).zip
	unzip -d bin -o bin/alp_$(OS)_$(ARCH).zip
	rm bin/alp_$(OS)_$(ARCH).zip
	chmod +x bin/alp

bin/pt-query-digest: bin
	wget -N -P bin percona.com/get/pt-query-digest
	chmod +x bin/pt-query-digest

log:
	mkdir log

bin:
	mkdir bin
