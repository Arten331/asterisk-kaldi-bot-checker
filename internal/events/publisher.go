package events

import (
	"context"
)

type EventPublisher struct {
	handlers map[string][]EventHandler
}

func NewEventPublisher() EventPublisher {
	return EventPublisher{
		handlers: map[string][]EventHandler{},
	}
}

func (e *EventPublisher) Subscribe(handler EventHandler, events ...Event) {
	for _, event := range events {
		handlers := e.handlers[event.Name()]
		handlers = append(handlers, handler)
		e.handlers[event.Name()] = handlers
	}
}

func (e *EventPublisher) Notify(ctx context.Context, event Event) {
	for _, handler := range e.handlers[event.Name()] {
		handler.Notify(ctx, event)
	}
}
