package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/tumbleweedd/two_services_system/order_service/internal/app/http"
	"github.com/tumbleweedd/two_services_system/order_service/internal/cache_impl"
	"github.com/tumbleweedd/two_services_system/order_service/internal/config"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"github.com/tumbleweedd/two_services_system/order_service/internal/repository"
	orderCancellationsService "github.com/tumbleweedd/two_services_system/order_service/internal/services/order/cancel"
	orderCreationService "github.com/tumbleweedd/two_services_system/order_service/internal/services/order/create"
	orderRetrievalService "github.com/tumbleweedd/two_services_system/order_service/internal/services/order/get"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/databases/postgres"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

func Run() {
	cfg := config.InitConfig()

	log := logger.SetupLogger(cfg.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := setupDatabase(ctx, log, &cfg)

	repo := repository.NewRepository(log, db.GetDB())

	cache := setupCache(log)

	orderCreationSvc := orderCreationService.New(log, repo)
	orderRetrievalSvc := orderRetrievalService.New(log, cache, repo)
	orderCancellationsSvc := orderCancellationsService.New(log, cache, repo, repo)

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

func setupDatabase(ctx context.Context, log *slog.Logger, cfg *config.Config) *postgres.PgDB {
	postgresDB, err := postgres.NewPostgresDB(ctx, log, postgresDSN(&cfg.Postgres))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to postgres: %v", err))
	}

	return postgresDB
}

func setupCache(log *slog.Logger) *cache_impl.Cache {
	hashicorpCache := expirable.NewLRU[uuid.UUID, *models.Order](5, nil, time.Minute*10)

	return cache_impl.NewCache(hashicorpCache, log)
}
