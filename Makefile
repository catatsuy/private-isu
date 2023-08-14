ca:
	git add .
	aicommits
	git push origin head
start:
	cd webapp && docker-compose up -d && cd ..
stop:
	cd webapp && docker-compose down && cd ..
restart:
	cd webapp && docker-compose down && docker-compose up -d && cd ..
bench:
	docker run --network host -i private-isu-benchmarker /opt/go/bin/benchmarker -t http://host.docker.internal -u /opt/go/userdata
mysql:
	cd webapp && docker-compose exec mysql bash
nginx:
	cd webapp && docker-compose exec nginx bash
rotate_nginx:
	cd webapp && docker-compose exec nginx bash -c "mv /var/log/nginx/access.log /var/log/nginx/access.log.$$(date +%Y%m%d%H%M%S)" && cd ..
	cd webapp && docker-compose exec nginx bash -c "nginx -s reopen" && cd ..
alp:
	cat webapp/logs/nginx/access.log | alp json --sort=sum -r -m "/image/[0-9]+\.(jpg|png|gif),/posts/[0-9]+,/@[a-z]"
ctop:
	ctop -a
