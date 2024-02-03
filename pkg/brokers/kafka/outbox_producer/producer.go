package outbox_producer

import (
	"fmt"
	"github.com/IBM/sarama"
	"log"
)

func NewProducer(port string) sarama.SyncProducer {
	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{fmt.Sprintf("localhost:%s", port)}, cfg)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	return producer
}
