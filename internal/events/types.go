package events

import (
	"context"

	"github.com/segmentio/kafka-go"

	kafkaClient "github.com/Arten331/messaging/kafka"
)

type Event interface {
	Name() string
}

type QueueableEvent interface {
	kafkaClient.QueueableMessage
	FromMessage(msg kafka.Message) (Event, error)
}

type EventHandler interface {
	Notify(ctx context.Context, event Event)
}
