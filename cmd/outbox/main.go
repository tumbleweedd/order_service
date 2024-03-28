package main

import (
	"context"
	"fmt"
	"github.com/tumbleweedd/two_services_system/order_service/internal/config"
	"github.com/tumbleweedd/two_services_system/order_service/internal/outbox_producer"
	producer "github.com/tumbleweedd/two_services_system/order_service/pkg/brokers/kafka/outboxProducer"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/databases/postgres"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

func main() {
	cfg := config.InitConfig()

	log := logger.NewSlogLogger(logger.SlogEnvironment(cfg.Env))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := postgres.NewPostgresDB(ctx, log, postgresDSN(&cfg.Postgres))
	if err != nil {
		panic(fmt.Sprintf("failed connect to db: %v", err.Error()))
	}

	newProducer := producer.NewProducer(cfg.Kafka.Port, log)

	outboxProducer := outbox_producer.New(newProducer, db.GetDB(), cfg.Kafka, log)

	if err = outboxProducer.ProduceMessages(ctx); err != nil {
		panic(fmt.Sprintf("produce messages error: %v", err.Error()))
	}

	log.Info("messages were successfully sent to their topics")
}

func postgresDSN(psqlCfg *config.PostgresConfig) string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		psqlCfg.Host, psqlCfg.Port, psqlCfg.User, psqlCfg.DbName, psqlCfg.Pwd, psqlCfg.SslMode)
}
