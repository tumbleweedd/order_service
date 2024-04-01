package main

import (
	"context"
	"fmt"
	outBoxRepository "github.com/tumbleweedd/two_services_system/order_service/internal/repository/outBox"

	"github.com/tumbleweedd/two_services_system/order_service/internal/config"
	outBoxService "github.com/tumbleweedd/two_services_system/order_service/internal/services/outBox/send"
	producer "github.com/tumbleweedd/two_services_system/order_service/pkg/brokers/kafka/producer"
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

	outBoxRepo := outBoxRepository.New(log, db.GetDB())

	outBoxSvc := outBoxService.New(log, cfg.Kafka, newProducer, outBoxRepo, outBoxRepo)

	if err = outBoxSvc.Send(ctx); err != nil {
		panic(fmt.Sprintf("produce messages error: %v", err.Error()))
	}

	log.Info("messages were successfully sent to their topics")
}

func postgresDSN(psqlCfg *config.PostgresConfig) string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		psqlCfg.Host, psqlCfg.Port, psqlCfg.User, psqlCfg.DbName, psqlCfg.Pwd, psqlCfg.SslMode)
}
