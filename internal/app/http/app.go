package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/config"
	cancelHandler "github.com/tumbleweedd/two_services_system/order_service/internal/delivery/http/order/cancel"
	createHandler "github.com/tumbleweedd/two_services_system/order_service/internal/delivery/http/order/create"
	getHandler "github.com/tumbleweedd/two_services_system/order_service/internal/delivery/http/order/get"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
)

type orderCreation interface {
	Create(ctx context.Context, order *models.Order) (string, error)
}

type orderCancellations interface {
	Cancel(ctx context.Context, orderUUID uuid.UUID) error
}

type orderRetrieval interface {
	OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) ([]models.Order, error)
	OrderByUUID(ctx context.Context, orderUUID uuid.UUID) (*models.Order, error)
}

type App struct {
	log        *slog.Logger
	httpServer *http.Server
}

func NewApp(
	log *slog.Logger,
	orderCreationSvc orderCreation,
	orderRetrievalSvc orderRetrieval,
	orderCancellationsSvc orderCancellations,
	cfg *config.HTTPConfig,
) *App {
	mux := chi.NewRouter()

	cancelH := cancelHandler.NewHandler(log, orderCancellationsSvc)
	createH := createHandler.NewHandler(log, orderCreationSvc)
	getH := getHandler.NewHandler(log, orderRetrievalSvc)

	mux.Route("/order", func(r chi.Router) {
		r.Post("/cancel", cancelH.Cancel)
		r.Post("/", createH.Create)
		r.Get("/", getH.OrdersByUUIDs)
	})

	httpServer := &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", cfg.Port),
	}

	return &App{
		log:        log,
		httpServer: httpServer,
	}
}

func (a *App) RunWithPanic() {
	if err := a.run(); err != nil {
		panic(fmt.Sprintf("failed to run http server: %v", err))
	}
}

func (a *App) run() error {
	a.log.With(slog.String("port", a.httpServer.Addr)).Info("starting http server")

	if err := a.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		a.log.Error("failed to run http server", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	log := a.log.With(slog.String("port", a.httpServer.Addr))

	log.Info("shutting down http server")

	return a.httpServer.Shutdown(ctx)
}
