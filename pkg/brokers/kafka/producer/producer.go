package producer

import (
	"fmt"

	"github.com/IBM/sarama"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

func NewProducer(port string, log logger.Logger) sarama.SyncProducer {
	const op = "producer.producer.NewProducer"

	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{fmt.Sprintf("localhost:%s", port)}, cfg)
	if err != nil {
		log.Error(op, logger.String("Failed to start Sarama producer:", err.Error()))
	}

	return producer
}
