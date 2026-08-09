Your Local folder for main project path:

cd /mnt/c/Users/DKGOSQLDT/gpu-enabled-minikube/dev-local/golang-k8s-microservices


Step Pre - A:
	cd init-services
	ENV_FILE="/mnt/c/Users/DKGOSQLDT/gpu-enabled-minikube/dev-local/golang-k8s-microservices/.env.local" source ./setports.sh

	env | grep -E 'MYSQL_DSN:ORDER_PORT|PORT|DATABASE_URL|MYSQL_DSN|REDIS_ADDR|RABBITMQ_URL'

Step Pre - B: one time
Volume:
	cd init-services/deploy/local
	chmod +x create-volumes.sh
	./create-volumes.sh

	start all services: i.e. [MYSQL|REDIS|RABBITMQ|POSTGRES]
	file: docker-compose.yml
	docker compose up -d

Step Pre - C: dbeaver already installed then
	dbeaver -nosplash -task "runningDBeaver"

Step POST - D: start order-service
	cd order-service
	go run cmd/api/main.go

Step POST - E: start inventory-service
	cd inventory-service
	go run cmd/api/main.go

Step Check Redis: browser
	docker run -d --name redisinsight -p 5540:5540 redis/redisinsight:latest
	