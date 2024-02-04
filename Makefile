.PHONY: init
init: webapp/sql/dump.sql
	$(MAKE) setup-bench-image

webapp/sql/dump.sql:
	cd webapp/sql && \
	curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/dump.sql.bz2 && \
	bunzip2 dump.sql.bz2

.PHONY: setup-bench-image
setup-bench-image:
	cd benchmarker/userdata && \
	curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/img.zip && \
	unzip img.zip
