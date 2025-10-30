# Common setup operations for nodes
docker-up: 
	docker compose -f deployment/docker/docker-compose.yml -p haqq-localnet up -d --build

docker-down:
	docker compose -f deployment/docker/docker-compose.yml -p haqq-localnet down --volumes 

docker-clean:
	docker compose -f deployment/docker/docker-compose.yml -p haqq-localnet down --volumes --rmi all