package main

import (
	"context"
	"fmt"
	"github.com/tumbleweedd/two_services_system/order_service/internal/app"
	"github.com/tumbleweedd/two_services_system/order_service/internal/config"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.InitConfig()

	log := logger.SetupLogger(cfg.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application, err := app.NewApp(ctx, log, &cfg, postgresDSN(&cfg.Postgres))
	if err != nil {
		panic(fmt.Sprintf("failed to create app: %v", err))
	}

	go application.HTTPServer.RunWithPanic()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	if err = application.Stop(); err != nil {
		panic(fmt.Sprintf("failed to stop app: %v", err))
	}

	log.Info("application stopped")
}

func postgresDSN(psqlCfg *config.PostgresConfig) string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		psqlCfg.Host, psqlCfg.Port, psqlCfg.User, psqlCfg.DbName, psqlCfg.Pwd, psqlCfg.SslMode)
}
