.PHONY: init
init: webapp/sql/dump.sql.bz2 benchmarker/userdata/img

webapp/sql/dump.sql.bz2:
	cd webapp/sql && \
	curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/dump.sql.bz2

benchmarker/userdata/img.zip:
	cd benchmarker/userdata && \
	curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/img.zip

benchmarker/userdata/img: benchmarker/userdata/img.zip
	cd benchmarker/userdata && \
	unzip -qq -o img.zip

start:
	cd webapp && docker compose up -d && sleep 5 && cd sql && bunzip2 -c dump.sql.bz2 | mysql -h 127.0.0.1 -P 3306 -u root -proot isuconp


bench:
	docker run --network host --add-host host.docker.internal:host-gateway -i private-isu-benchmarker /bin/benchmarker -t http://host.docker.internal -u /opt/userdata