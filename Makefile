up:
	docker-compose -f docker/docker-compose.yml build app
	docker-compose -f docker/docker-compose.yml up -d

down:
	docker-compose -f docker/docker-compose.yml down
