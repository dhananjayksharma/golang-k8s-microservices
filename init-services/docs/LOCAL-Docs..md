golang-k8s-microservices:
Step 1:
	minikube start   --driver=docker   --container-runtime=docker   --gpus=all   --cpus=4   --memory=6144   --disk-size=40g



order-service start
Step 2:
	cd order-service/deploy/local
	docker compose up -d
	docker compose ps
	cd ../..			cd order-service
	export DATABASE_URL='postgres://order_user:order_dummy@localhost:5432/order_db?sslmode=disable'
	migrate -path migrations-dbs/order-data -database "$DATABASE_URL" up

	go run .


inventory-service start
Step 3:
	cd inventory-service/deploy/local
	docker compose up -d
	docker compose ps
	cd ../..  		cd inventory-service
	
	export MYSQL_DSN='root:root@tcp(localhost:3306)/appdb?parseTime=true'
	export MYSQL_MIGRATE_URL='mysql://root:root@tcp(localhost:3306)/appdb?multiStatements=true' 
	
	migrate \
		-path migrations-dbs/inventory-data \
		-database "$MYSQL_MIGRATE_URL" \
	up
	
	docker compose -f deploy/local/docker-compose.yml exec -T mysql \
	mysql -uroot -proot appdb < migrations/001_inventory_seed.sql
  
	migrate -path migrations-dbs/order-data -database "$MYSQL_DSN" up
	go run cmd/api/main.go
	
Step 3: DBeaver
	docker inspect local-postgres-1 --format '{{range .Config.Env}}{{println .}}{{end}}' |
	grep POSTGRES
	POSTGRES_USER=order_user
	POSTGRES_PASSWORD=order_dummy
	POSTGRES_DB=order_db
	PORT_DB=5432

Step 4: migration

check docs:
cd order-service
go generate ./internal/openapi

docker run --rm \
--name order-swagger-ui \
-p 8088:8080 \
-e SWAGGER_JSON=/spec/openapi.yaml \
-v "$PWD/order-service/cmd/api/openapi.yaml:/spec/openapi.yaml:ro" \
swaggerapi/swagger-ui
  
cd /init-services/deploy/local
docker compose -f swagger-compose.yml up -d --force-recreate

