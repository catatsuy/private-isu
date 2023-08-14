ca:
	git add .
	aicommits
	git push origin head
start:
	cd webapp && docker-compose up -d && cd ..
stop:
	cd webapp && docker-compose down && cd ..
