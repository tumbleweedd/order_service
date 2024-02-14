package httpapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	order_service_http "github.com/tumbleweedd/two_services_system/order_service/internal/delivery/http"
)

type App struct {
	log        *slog.Logger
	httpServer *http.Server
	port       int
}

func NewApp(log *slog.Logger, orderService order_service_http.Order, port int) *App {
	handler := order_service_http.NewHandler(log, orderService)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler.InitRoutes(),
	}

	return &App{
		log:        log,
		httpServer: httpServer,
		port:       port,
	}
}

func (a *App) RunWithPanic() {
	if err := a.Run(); err != nil {
		panic(fmt.Sprintf("failed to run http server: %v", err))
	}
}

func (a *App) Run() error {
	const op = "httpapp.run"

	log := a.log.With(slog.String("op", op), slog.Int("port", a.port))

	log.Info("starting http server")

	return a.httpServer.ListenAndServe()
}

func (a *App) Stop() error {
	const op = "httpapp.stop"

	log := a.log.With(slog.String("op", op))

	log.Info("stopping http server")

	return a.httpServer.Shutdown(context.Background())
}
