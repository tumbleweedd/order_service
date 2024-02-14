package outbox_producer

import (
	"fmt"
	"github.com/IBM/sarama"
	"log/slog"
)

func NewProducer(port string, log *slog.Logger) sarama.SyncProducer {
	const op = "producer.producer.NewProducer"

	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{fmt.Sprintf("localhost:%s", port)}, cfg)
	if err != nil {
		log.Error(op, slog.String("Failed to start Sarama producer:", err.Error()))
	}

	return producer
}
