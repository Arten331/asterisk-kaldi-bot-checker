package botchecker

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Arten331/bot-checker/data/embed"
	"github.com/Arten331/bot-checker/internal/botchecker/metrics"
	"github.com/Arten331/bot-checker/internal/domain/phrase"
	"github.com/Arten331/bot-checker/internal/events"
	"github.com/Arten331/bot-checker/internal/models"
	"github.com/Arten331/bot-checker/pkg/kaldi"
	"github.com/Arten331/observability/logger"
	"github.com/CyCoreSystems/ari"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type MetricService interface {
	Register(prometheus.Collector) error
	AddMiddleware(func(handler http.Handler) http.Handler)
}

type Options struct {
	EventPublisher        events.EventPublisher
	MetricService         MetricService
	StopPhrasesRepository phrase.Repository
	KaldiClient           *kaldi.Client
	AriClient             ari.Client
	SaveRecords           bool
}

type BotChecker struct {
	EventPublisher        events.EventPublisher
	Metrics               metrics.Metrics
	stopPhrasesRepository phrase.Repository
	KaldiClient           *kaldi.Client
	AriClient             ari.Client
	SaveRecords           bool
}

func New(o *Options) (*BotChecker, error) {
	botChecker := BotChecker{
		stopPhrasesRepository: o.StopPhrasesRepository,
		KaldiClient:           o.KaldiClient,
		AriClient:             o.AriClient,
		EventPublisher:        o.EventPublisher,
		Metrics: metrics.Metrics{
			Service: o.MetricService,
		},
	}

	if botChecker.stopPhrasesRepository == nil {
		return nil, errors.New("service botchecker require StopPhrasesRepository")
	}

	if botChecker.KaldiClient == nil {
		return nil, errors.New("service botchecker require KaldiClient")
	}

	botChecker.Metrics.Register()
	//nolint:gocritic // example, how add middleware to prometheus
	/*botChecker.metrics.service.AddMiddleware(func(handler http.Handle) http.Handle {
		fn := func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
			botChecker.metrics.ResetNoiseHangup()
		}

		return http.HandlerFunc(fn)
	})*/

	return &botChecker, nil
}

func (b *BotChecker) Run(_ context.Context, cancelFunc context.CancelFunc) {
	err := b.loadStopPhrases()
	if err != nil {
		logger.L().Error("failed run botchecker service", zap.Error(err))

		cancelFunc()
	}

	if b.AriClient != nil {
		info, err := b.AriClient.Asterisk().Info(nil)
		if err != nil {
			logger.L().Error("Failed to get Asterisk Info", zap.Error(err))

			return
		}

		logger.L().Info("Asterisk Info", zap.Reflect("info", info))
	}
}

func (b *BotChecker) loadStopPhrases() error {
	tfs := embed.GetEmbedFilesystem()

	phrasesFile, err := tfs.Open("records_mini.csv")
	if err != nil {
		return err
	}

	reader := csv.NewReader(phrasesFile)

	phrases := make([]*phrase.StopPhrase, 0)

	var row []string

	for {
		row, err = reader.Read()
		if err == io.EOF || cap(row) == 1 {
			break
		}

		if row[0] != "" && row[1] != "" {
			phrases = append(phrases, phrase.New(row[0], row[1]))
		}
	}

	err = b.stopPhrasesRepository.Load(phrases)
	if err != nil {
		return err
	}

	return nil
}

func (b *BotChecker) Check(
	ctx context.Context,
	cancel context.CancelFunc,
	mshCh <-chan models.KaldiMessage,
	errCh chan error,
) (isBot bool, stopPhrase *phrase.StopPhrase, err error) {
	var firstWordRead bool

	for {
		select {
		case <-ctx.Done():
			return false, stopPhrase, nil
		case msg := <-mshCh:
			logger.L().Debug("kaldi received", zap.ByteString("msg", msg.Text))

			if !firstWordRead {
				firstWordRead = bytes.Contains(msg.Text, []byte{' '})

				if !firstWordRead {
					continue
				}
			}

			isPartial, isBot, resPhrase := b.phraseBelongsToTheBot(msg)

			switch {
			case isBot == true:
				return true, resPhrase, nil
			case isPartial == true:
				continue
			}

			if msg.IsFinal { // if first phrase short try check next
				firstWordRead = false

				continue
			}

			logger.L().Debug("ivr stop phrase not found", zap.Object("msg", msg))

			return false, stopPhrase, nil
		case err = <-errCh:
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logger.L().Info("stop recognition - canceled")
			} else {
				logger.L().Error("", zap.Error(err))
			}

			cancel()

			return false, stopPhrase, err
		}
	}
}

func (b *BotChecker) phraseBelongsToTheBot(msg models.KaldiMessage) (isPartial, isBot bool, p *phrase.StopPhrase) {
	search := msg.Text

	if !msg.IsFinal {
		lastSpaceIndex := bytes.LastIndexByte(msg.Text, ' ')
		if lastSpaceIndex > 0 {
			search = msg.Text[:lastSpaceIndex]
		}
	}

	res, err := b.stopPhrasesRepository.FindCloser(string(search))
	if err != nil {
		return false, false, nil
	}

	switch {
	case strings.HasPrefix(string(search), res.Phrase):
		return true, true, res
	case strings.Contains(res.Phrase, string(search)):
		return true, false, res
	}

	return false, false, nil
}
