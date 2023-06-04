//go:build test && integration

package kaldi

import (
	"context"
	"errors"
	"testing"

	testdata "github.com/Arten331/bot-checker/data/test"
	"github.com/Arten331/bot-checker/internal/config"
	"github.com/Arten331/observability/logger"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type FilesTestCase struct {
	desc     string
	record   string
	expected string
}

func TestClient(t *testing.T) {
	cfg, err := config.Init()
	require.NoError(t, err)

	testFS := testdata.GetTestFS()

	client := NewClient(Options{
		Host: cfg.Kaldi.Host,
		Port: cfg.Kaldi.Port,
	})

	tcases := []FilesTestCase{
		{
			record:   "20220216_SUBSCRIBER_NOT_AVAIL.wav",
			expected: "абонент временно недоступен попробуйте позвонить позже",
		},
		{
			record:   "20220217_BLOCKED.wav",
			expected: "извините набранный вами номер временно заблокирован",
		},
		{
			record:   "20220217_BUSY_WAITING.wav",
			expected: "пожалуйста оставайтесь на линии сейчас абонент разговаривает",
		},
		{
			record:   "20220217_DISCONNECTED1.wav",
			expected: "телефон абонента выключен или находится вне зоны обслуживания",
		},
	}

	for i := range tcases {
		func(tc FilesTestCase) {
			t.Run(tc.record, func(t *testing.T) {
				t.Parallel()

				file, err := testFS.Open("records/" + tc.record)
				require.NoError(t, err)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				resCh, errCh := client.ProcessAudio(ctx, file)
				for {
					select {
					case res := <-resCh:
						logger.L().Debug("read message", zap.Object("message", res))

						if res.IsFinal {
							logger.L().Info("End voice recognition", zap.Object("message", res))
							require.Contains(t, string(res.Text), tc.expected)

							return
						}
					case err = <-errCh:
						if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
							logger.L().Info("stop recognition - canceled")
						} else {
							logger.L().Error("", zap.Error(err))
						}

						t.Error(err)

						cancel()

						return
					}
				}
			})
		}(tcases[i])

	}
}
