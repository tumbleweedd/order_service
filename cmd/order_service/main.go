package main

import (
	"context"
	"fmt"
	"github.com/tumbleweedd/two_services_system/order_service/internal/app"
	"github.com/tumbleweedd/two_services_system/order_service/internal/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
)

func main() {
	cfg := config.InitConfig()

	log := setupLogger(cfg.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application, err := app.NewApp(ctx, log, cfg.HTTP.Port, cfg.Kafka.OrderEventTopic, cfg.Kafka.StatusEventTopic, postgresDSN(&cfg.Postgres), cfg.Kafka.BrokerList)
	if err != nil {
		log.Error(fmt.Sprintf("failed to create app: %v", err))
	}

	go application.HTTPServer.RunWithPanic()

	go application.Producer.ProduceOrderEvent(ctx)
	go application.Producer.ProduceStatusEvent(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	if err = application.Stop(); err != nil {
		log.Error(fmt.Sprintf("failed to stop app: %v", err))
	}

	log.Info("application stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	}

	return log
}

//func startProducers(ctx context.Context, application *app.App, cfg *config.KafkaConfig) {
//	go application.Producer.ProduceOrderEvent(ctx, cfg.OrderEventTopic)
//	go application.Producer.ProduceStatusEvent(ctx, cfg.StatusEventTopic)
//}

func postgresDSN(psqlCfg *config.PostgresConfig) string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		psqlCfg.Host, psqlCfg.Port, psqlCfg.User, psqlCfg.DbName, psqlCfg.Pwd, psqlCfg.SslMode)
}
