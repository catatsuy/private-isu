.PHONY: pre-bench bench post-bench

bench:
	docker run --network host -i private-isu-benchmarker /opt/go/bin/benchmarker -t http://host.docker.internal -u /opt/go/userdata

pre-bench:
	truncate -s 0  logs/nginx/access.log
	truncate -s 0  logs/nginx/error.log
	truncate -s 0  logs/mysql/slow-query.log
	docker stats

post-bench:
	mkdir -p bench/${V}
	cat logs/nginx/access.log | alp ltsv -m="^/posts/[0-9]+","^/image/[0-9]+\.(jpg|png|gif)","^/@[a-z]*" --reverse > bench/${V}/alp.log
	pt-query-digest logs/mysql/slow-query.log > bench/${V}/slow-query.log