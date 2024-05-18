package send

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/tumbleweedd/two_services_system/order_service/internal/config"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

type outBoxGetter interface {
	FetchUnprocessedMessages(ctx context.Context) (messages []models.OutBoxMessage, err error)
}

type outBoxRemover interface {
	Delete(ctx context.Context, eventIDs []int) error
}

type Service struct {
	log              logger.Logger
	kafkaConfig      config.KafkaConfig
	producer         sarama.SyncProducer
	messageProcessor outBoxGetter
	outBoxRemover    outBoxRemover
}

func New(
	log logger.Logger,
	kafkaConfig config.KafkaConfig,
	producer sarama.SyncProducer,
	outBpxGetter outBoxGetter,
	outBoxRemover outBoxRemover,
) *Service {
	return &Service{
		log:              log,
		kafkaConfig:      kafkaConfig,
		producer:         producer,
		messageProcessor: outBpxGetter,
		outBoxRemover:    outBoxRemover,
	}
}

func (s *Service) Send(ctx context.Context) error {
	messages, err := s.messageProcessor.FetchUnprocessedMessages(ctx)
	if err != nil {
		return fmt.Errorf("fetch unprocessed messages: %w", err)
	}

	saramaMessages := make([]*sarama.ProducerMessage, 0, len(messages))
	processedMessagesIDs := make([]int, 0, len(messages))

	for _, msg := range messages {
		saramaMessages = append(saramaMessages, &sarama.ProducerMessage{
			Topic: s.kafkaConfig.OrderEventTopic,
			Value: sarama.ByteEncoder(msg.Payload),
		})

		processedMessagesIDs = append(processedMessagesIDs, msg.ID)
	}

	if err = s.producer.SendMessages(saramaMessages); err != nil {
		return fmt.Errorf("send messages: %w", err)
	}

	if err = s.outBoxRemover.Delete(ctx, processedMessagesIDs); err != nil {
		return fmt.Errorf("remove messages: %w", err)
	}

	return nil
}
