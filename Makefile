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

COMPOSE := docker compose -f webapp/compose.yml
BENCH_TARGET ?= http://localhost

# 計測ログを空にする。ログは追記されるので、計測のたびに実行しないと
# 前回の結果が混ざったまま alp / pt-query-digest に食わせることになる。
# 単に truncate するだけでなくログを開き直させる (nginx -s reopen /
# slow_query_log の OFF→ON) ことで、プロセスが掴んだままの fd を捨てる。
.PHONY: log-clean
log-clean:
	$(COMPOSE) exec -T nginx sh -c ': > /var/log/nginx/access.log; : > /var/log/nginx/error.log; nginx -s reopen'
	$(COMPOSE) exec -T mysql sh -c 'mysql -uroot -p"$$MYSQL_ROOT_PASSWORD" -e "SET GLOBAL slow_query_log = OFF" && : > /var/log/mysql/slow.log && mysql -uroot -p"$$MYSQL_ROOT_PASSWORD" -e "SET GLOBAL slow_query_log = ON"'

# ログを消してからベンチを回す。計測はこれ経由で行う
.PHONY: bench
bench: log-clean
	cd benchmarker && ./bin/benchmarker -t "$(BENCH_TARGET)" -u ./userdata
