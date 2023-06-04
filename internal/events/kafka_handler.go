package events

import (
	"context"
	"errors"

	"github.com/Arten331/messaging/kafka"
	"github.com/Arten331/observability/logger"
	"go.uber.org/zap"
)

var errKafkaHandlerWrongEventType = errors.New(
	"unable send event to kafka, wrong type, type assertion failed(kafka.QueueableMessage)",
)

type KafkaEventHandler struct {
	producer kafka.Producer
}

func (k KafkaEventHandler) Notify(ctx context.Context, event Event) {
	msg, isQueueable := event.(kafka.QueueableMessage)
	if !isQueueable {
		logger.L().Error(errKafkaHandlerWrongEventType.Error(), zap.String("event", event.Name()))

		return
	}

	cMessages := []kafka.QueueableMessage{msg}

	err := k.producer.SendMessages(ctx, cMessages)
	if err != nil {
		logger.L().Error("Unable send event to kafka", zap.String("event", event.Name()))
	}
}

func NewKafkaEventHandler(p kafka.Producer) KafkaEventHandler {
	return KafkaEventHandler{
		producer: p,
	}
}
