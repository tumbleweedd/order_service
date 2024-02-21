## How to run
```
 // to run postgres, kafka
docker-compose up -d

// to run app
go run cmd/order_service/main.go --config=config/config.yaml

// to run outbox
go run cmd/outbox/main.go --config=config/config.yaml

// to run migrations
go run cmd/migrator/main.go -storage-path "postgres:postgres@localhost:5432/order_service?sslmode=disable" -migrations-path migrations
```
