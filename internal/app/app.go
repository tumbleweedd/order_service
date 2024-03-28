package app

import (
	"context"
	"fmt"
	"github.com/tumbleweedd/two_services_system/order_service/internal/app/http"
	"github.com/tumbleweedd/two_services_system/order_service/internal/cacheImpl"
	"github.com/tumbleweedd/two_services_system/order_service/internal/config"
	orderRepository "github.com/tumbleweedd/two_services_system/order_service/internal/repository/order"
	outBoxRepository "github.com/tumbleweedd/two_services_system/order_service/internal/repository/outBox"
	orderCancellationsService "github.com/tumbleweedd/two_services_system/order_service/internal/services/order/cancel"
	orderCreationService "github.com/tumbleweedd/two_services_system/order_service/internal/services/order/create"
	orderRetrievalService "github.com/tumbleweedd/two_services_system/order_service/internal/services/order/get"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/databases/postgres"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run() {
	cfg := config.InitConfig()

	log := logger.NewSlogLogger(logger.SlogEnvironment(cfg.Env))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := setupDatabase(ctx, log, &cfg)

	outBoxRepo := outBoxRepository.New(log, db.GetDB())
	orderRepo := orderRepository.NewOrderRepository(log, db.GetDB(), outBoxRepo)

	cache := cacheImpl.NewCache(ctx, 10*time.Minute)

	orderCreationSvc := orderCreationService.New(log, cache, orderRepo)
	orderRetrievalSvc := orderRetrievalService.New(log, cache, orderRepo)
	orderCancellationsSvc := orderCancellationsService.New(log, cache, orderRepo, orderRepo)

	httpServer := http.NewApp(
		log,
		orderCreationSvc,
		orderRetrievalSvc,
		orderCancellationsSvc,
		&cfg.HTTP,
	)

	go func() {
		httpServer.RunWithPanic()
	}()

	log.Info("http server started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	log.Info("stopping http server")

	if err := httpServer.Shutdown(ctx); err != nil {
		panic(fmt.Sprintf("failed to shutdown http server: %v", err))
	}

	log.Info("http server stopped")

	if err := db.Close(); err != nil {
		panic(fmt.Sprintf("failed to close postgres: %v", err))
	}

	log.Info("postgres db closed")

}

func postgresDSN(psqlCfg *config.PostgresConfig) string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		psqlCfg.Host, psqlCfg.Port, psqlCfg.User, psqlCfg.DbName, psqlCfg.Pwd, psqlCfg.SslMode)
}

func setupDatabase(ctx context.Context, log logger.Logger, cfg *config.Config) *postgres.PgDB {
	postgresDB, err := postgres.NewPostgresDB(ctx, log, postgresDSN(&cfg.Postgres))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to postgres: %v", err))
	}

	return postgresDB
}
