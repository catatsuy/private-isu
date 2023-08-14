ca:
	git add .
	aicommits
	git push origin head
start:
	cd webapp && docker-compose up -d && cd ..
stop:
	cd webapp && docker-compose down && cd ..
bench:
	docker run --network host -i private-isu-benchmarker /opt/go/bin/benchmarker -t http://host.docker.internal -u /opt/go/userdata
mysql:
	docker-compose exec mysql bash
