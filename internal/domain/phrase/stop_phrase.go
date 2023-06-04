package phrase

import (
	"go.uber.org/zap/zapcore"
)

type Category []byte

func (c Category) Name() string {
	return string(c)
}

type StopPhrase struct {
	Phrase   string   `json:"phrase,omitempty"`
	Category Category `json:"category,omitempty"`
}

func New(phrase, category string) *StopPhrase {
	return &StopPhrase{
		Phrase:   phrase,
		Category: Category(category),
	}
}

func (p *StopPhrase) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("phrase", p.Phrase)
	encoder.AddString("category", string(p.Category))

	return nil
}
