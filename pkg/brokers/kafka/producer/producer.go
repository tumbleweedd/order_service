package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"log/slog"
)

type Producer struct {
	log *slog.Logger

	orderEventTopic  string
	statusEventTopic string

	orderEventsChan       chan models.Event
	changeStatusEventChan chan models.Event
	done                  chan struct{}

	// выполнение дальнейших действий в сервисе заказов не зависит от успешной
	// обработки заказа брокером, поэтому использую sarama.AsyncProducer
	producer sarama.AsyncProducer
}

func NewProducer(
	ctx context.Context,
	log *slog.Logger,
	orderEventTopic string,
	statusEventTopic string,
	orderEventsChan chan models.Event,
	changeStatusEventChan chan models.Event,
	done chan struct{},
	brokerAddress []string,
) (*Producer, error) {
	producerConfig := sarama.NewConfig()
	//config.Producer.Idempotent = true
	//config.Net.MaxOpenRequests = 1
	producerConfig.Producer.RequiredAcks = sarama.WaitForLocal
	producerConfig.Producer.Compression = sarama.CompressionNone
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.Return.Errors = true

	producer, err := sarama.NewAsyncProducer(brokerAddress, producerConfig)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case sendErr, ok := <-producer.Errors():
				if !ok {
					return
				}

				log.Warn("failed to send message", slog.String("error", sendErr.Error()))
			case success, ok := <-producer.Successes():
				if !ok {
					return
				}

				log.Debug("successfully sent message", slog.String("topic", success.Topic))
			case <-ctx.Done():
				return
			}
		}
	}()

	return &Producer{
		log:                   log,
		orderEventsChan:       orderEventsChan,
		changeStatusEventChan: changeStatusEventChan,
		producer:              producer,
		orderEventTopic:       orderEventTopic,
		statusEventTopic:      statusEventTopic,
		done:                  done,
	}, nil
}

func (p *Producer) ProduceOrderEvent(ctx context.Context) {
	const op = "brokers.kafka.producer.ProduceOrderEvent"

	p.eventProcessing(ctx, op, p.orderEventTopic)
}

func (p *Producer) ProduceStatusEvent(ctx context.Context) {
	const op = "brokers.kafka.producer.ProduceStatusEvent"

	p.eventProcessing(ctx, op, p.statusEventTopic)
}

func (p *Producer) eventProcessing(ctx context.Context, op string, topic string) {
	var eventChan chan models.Event
	switch topic {
	case p.orderEventTopic:
		eventChan = p.orderEventsChan
	case p.statusEventTopic:
		eventChan = p.changeStatusEventChan
	}

ProducerLoop:
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				break ProducerLoop
			}

			p.log.Debug(op, fmt.Sprintf("send %s #%s to kafka", topic, event.UUID()))
			bytes, err := json.Marshal(event)
			if err != nil {
				p.log.Error(op, slog.String("failed to marshal order", err.Error()))
				continue
			}

			message := &sarama.ProducerMessage{
				Topic: topic,
				Key:   sarama.StringEncoder(event.UUID()),
				Value: sarama.ByteEncoder(bytes),
			}

			p.producer.Input() <- message
		case <-ctx.Done():
			break ProducerLoop
		}
	}
}

func (p *Producer) Close() error {
	close(p.producer.Input())

	err := p.producer.Close()
	if err != nil {
		return err
	}

	close(p.done)
	close(p.orderEventsChan)
	close(p.changeStatusEventChan)

	return nil
}
