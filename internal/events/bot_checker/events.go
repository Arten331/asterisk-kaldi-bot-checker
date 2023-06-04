package checkevents

import (
	"encoding/json"
	"time"

	"github.com/Arten331/bot-checker/internal/app/global"
	"github.com/Arten331/bot-checker/internal/events"
	kafkaClient "github.com/Arten331/messaging/kafka"
	"github.com/segmentio/kafka-go"
)

const (
	KeyBotFound          = "bot_checker_robot_found"
	KeyBotNotFound       = "bot_checker_robot_not_found"
	typeBotCheckerResult = "bot_checker_result"
)

type Event interface {
	events.Event
	kafkaClient.QueueableMessage
}

type ClickKafkaMessage struct {
	Timestamp   int64  `json:"timestamp"`
	App         string `json:"app"`
	Environment string `json:"environment"`
	Type        string `json:"type"`
	Data        string `json:"data"`
}

type BotFound struct {
	CallID    string `json:"id"`
	Dest      string `json:"dnid"`
	From      string `json:"from"`
	Phrase    string `json:"phrase"`
	EventName string `json:"event_name"`
}

func (e *BotFound) Name() string {
	return KeyBotFound
}

func (e *BotFound) KafkaMessage() (kafka.Message, error) {
	km := NewClickKafkaMessage(e)

	msg, err := json.Marshal(km)
	if err != nil {
		return kafka.Message{}, err
	}

	return kafka.Message{
		Key:   []byte(e.Name()),
		Value: msg,
	}, nil
}

type BotNotFounded struct {
	CallID    string `json:"id"`
	Dest      string `json:"dnid"`
	From      string `json:"from"`
	Phrase    string `json:"phrase"`
	EventName string `json:"event_name"`
}

func (e *BotNotFounded) Name() string {
	return KeyBotNotFound
}

func (e *BotNotFounded) KafkaMessage() (kafka.Message, error) {
	km := NewClickKafkaMessage(e)

	msg, err := json.Marshal(km)
	if err != nil {
		return kafka.Message{}, err
	}

	return kafka.Message{
		Key:   []byte(e.Name()),
		Value: msg,
	}, nil
}

func NewClickKafkaMessage(data any) ClickKafkaMessage {
	eventData, _ := json.Marshal(data)

	cm := ClickKafkaMessage{
		Timestamp:   time.Now().Unix(),
		App:         global.AppName(),
		Environment: global.AppEnv(),
		Type:        typeBotCheckerResult,
		Data:        string(eventData),
	}

	return cm
}
