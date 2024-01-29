package app

import (
	"context"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/brokers/kafka/producer"
	"log/slog"

	httpapp "github.com/tumbleweedd/two_services_system/order_service/internal/app/http"
	"github.com/tumbleweedd/two_services_system/order_service/internal/repository"
	"github.com/tumbleweedd/two_services_system/order_service/internal/services"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/databases/postgres"
)

type App struct {
	HTTPServer *httpapp.App
	PostgresDB *postgres.PgDB
	Producer   *producer.Producer
}

//type AppBuilder interface {
//	SetCtx(ctx context.Context) AppBuilder
//	SetLogger(logger *slog.Logger) AppBuilder
//	SetHttpPort(port int) AppBuilder
//	SetOrderEventTopic(topicName string) AppBuilder
//	SetStatusEventTopic(topicName string) AppBuilder
//	SetPostgresDSN(dsn string) AppBuilder
//	SetBrokerAddresses(addresses []int) AppBuilder
//	Build() *App
//}
//
//func (a *App) SetCtx(ctx context.Context) AppBuilder {
//	a.c
//}
//
//func (a *App) SetLogger(logger *slog.Logger) AppBuilder {
//
//}
//
//func (a *App) SetHttpPort(port int) AppBuilder {
//
//}
//
//func (a *App) SetOrderEventTopic(topicName string) AppBuilder {
//
//}
//
//func (a *App) SetStatusEventTopic(topicName string) AppBuilder {
//
//}
//
//func (a *App) SetPostgresDSN(dsn string) AppBuilder {
//
//}
//
////func (a *App) SetBrokerAddresses(addresses []int) AppBuilder AppBuilder {
////
////}

func NewApp(
	ctx context.Context,
	log *slog.Logger,
	httpPort int,
	orderEventTopicName string,
	statusEventTopicName string,
	postgresDSN string,
	brokerAddress []string,
) (*App, error) {
	postgresDB, err := postgres.NewPostgresDB(ctx, log, postgresDSN)
	if err != nil {
		log.Error("failed to connect to postgres", err)
		return nil, err
	}

	repo := repository.NewRepository(log, postgresDB.GetDB())

	orderEventsChan := make(chan models.Event, 1)
	statusEventChan := make(chan models.Event, 1)
	svc := services.NewService(log, repo, repo, orderEventsChan, statusEventChan)

	httpApp := httpapp.NewApp(log, svc, httpPort)

	orderProducer, err := producer.NewProducer(
		ctx,
		log,
		orderEventTopicName,
		statusEventTopicName,
		orderEventsChan, statusEventChan,
		brokerAddress)
	if err != nil {
		log.Error("failed to connect to kafka", err)
		return nil, err
	}

	return &App{
		HTTPServer: httpApp,
		PostgresDB: postgresDB,
		Producer:   orderProducer,
	}, nil
}

//func (a *App) Run() error {
//	err := a.HTTPServer.Run()
//	if err != nil {
//		return err
//	}
//
//	return nil
//}

func (a *App) Stop() error {
	err := a.HTTPServer.Stop()
	if err != nil {
		return err
	}

	if err = a.PostgresDB.Close(); err != nil {
		return err
	}

	if err = a.Producer.Close(); err != nil {
		return err
	}

	return nil
}
