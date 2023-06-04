package models

import "go.uber.org/zap/zapcore"

type KaldiMessage struct {
	Text    []byte
	IsFinal bool
}

func (k KaldiMessage) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("text", string(k.Text))
	encoder.AddBool("is_final", k.IsFinal)

	return nil
}
