package botchecker

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Arten331/bot-checker/internal/domain/phrase"
	checkevents "github.com/Arten331/bot-checker/internal/events/bot_checker"
	"github.com/Arten331/bot-checker/pkg/audio"
	commands2 "github.com/Arten331/bot-checker/pkg/audio/commands"
	"github.com/Arten331/observability/logger"
	"github.com/CyCoreSystems/ari"
	"github.com/go-chi/chi/v5"
	"github.com/gobwas/ws"
	"go.uber.org/zap"
)

func (b *BotChecker) CheckBotHandler() http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var err error

		uniqID := chi.URLParam(r, "uniqID")

		b.Metrics.StoreIvrCheckStart()

		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			logger.L().Error("handshake error", zap.Error(err))

			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*120)
		defer cancel()

		out, err := b.soxFlow(ctx, cancel, conn)
		if err != nil {
			logger.L().Error("Failed create audio pipe", zap.Error(err))

			cancel()

			return
		}

		defer func() {
			file, ok := out.(*os.File)
			if ok {
				_ = os.Remove(file.Name())
			}
		}()

		resCh, errCh := b.KaldiClient.ProcessAudio(ctx, out)

		isBot, stopPhrase, err := b.Check(ctx, cancel, resCh, errCh)
		if errors.Is(err, phrase.ErrPhraseNotFound) {
			logger.L().Info("bot is not finded", zap.Error(err))
		}

		if err != nil {
			logger.L().Info("err find stop phrase", zap.Error(err))

			cancel()

			return
		}

		if isBot {
			logger.L().Info("found a bot", zap.Object("phrase", stopPhrase))

			b.HangupBot(ctx, uniqID, stopPhrase)
			<-time.After(time.Second * 1)

			return
		}

		logger.L().Info("Bot not found")

		<-ctx.Done()

		return
	}

	return fn
}

func (b *BotChecker) HangupBot(ctx context.Context, uniqID string, stopPhrase *phrase.StopPhrase) {
	channel := b.AriClient.Channel().Get(&ari.Key{
		Kind: ari.ChannelKey,
		ID:   uniqID,
	})

	caller, _ := channel.GetVariable("CALLERID(num)")
	dnID, _ := channel.GetVariable("DNID")

	b.EventPublisher.Notify(ctx, &checkevents.BotFound{
		CallID:    uniqID,
		Dest:      dnID,
		From:      caller,
		Phrase:    stopPhrase.Phrase,
		EventName: checkevents.KeyBotFound,
	})

	err := channel.Hangup()
	if err != nil {
		logger.L().Error("Unable hangup channel", zap.Error(err))

		return
	}

	b.Metrics.StoreIvrCheckHangup(stopPhrase)

	logger.L().Info("bot hangup", zap.Object("phrase", stopPhrase))
}

func (b *BotChecker) soxFlow(ctx context.Context, cancel context.CancelFunc, conn net.Conn) (io.Reader, error) {
	var (
		err       error
		outResult io.Reader
		audioOut  io.Reader
		audioIn   io.Writer
	)

	audioOut, audioIn, err = os.Pipe()
	if err != nil {
		return nil, err
	}

	if b.SaveRecords {
		var tmpRaw io.Writer

		tmpRaw, err = os.CreateTemp("/tmp/botrec", "raw_"+time.Now().Format(time.RFC3339Nano)+"*.pcm")
		if err != nil {
			logger.L().Info("error create tmp pcm record", zap.Error(err))

			return nil, err
		}

		audioOut = io.TeeReader(audioOut, tmpRaw)
	}

	readAudioForkMessages(ctx, cancel, conn, audioIn)
	logger.L().Info("sox started")

	pipe, err := audio.NewPipe([]audio.PipeCommand{
		commands2.NewPcmRawToWav(),
		commands2.NewPcmWavSilence(),
	})
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				buf := make([]byte, audio.BUFFSIZE)

				_, err = io.ReadFull(audioOut, buf)
				if err == io.EOF {
					continue
				}

				err = pipe.Write(buf)

				if err != nil {
					logger.L().Error("failed write to pipe", zap.Error(err))

					cancel()

					return
				}
			}
		}
	}()

	err = pipe.Run(ctx)
	if err != nil {
		logger.L().Error("error run sox pipe", zap.Error(err))

		return nil, err
	}

	outResult = pipe.StdOut

	return outResult, nil
}

func readAudioForkMessages(ctx context.Context, cancel context.CancelFunc, conn net.Conn, inAudio io.Writer) {
	var (
		err        error
		soxStarted bool
		header     ws.Header
	)

	startSox := make(chan interface{}, 1)

	go func() {
		defer func() { _ = conn.Close() }()

		for {
			select {
			case <-ctx.Done():
				inAudio = io.Discard

				return
			default:
				header, err = ws.ReadHeader(conn)
				if err != nil {
					logger.L().Error("unable read ws header", zap.Error(err))
					cancel()

					return
				}

				if !soxStarted {
					startSox <- struct{}{}

					soxStarted = true
				}

				payload := make([]byte, header.Length)

				_, err = io.ReadFull(conn, payload)
				if err != nil {
					logger.L().Error("unable read ws message", zap.Error(err))
					cancel()

					return
				}

				_, _ = inAudio.Write(payload)

				if header.OpCode == ws.OpClose {
					logger.L().Debug("OpClose AudioFork")
					cancel()

					return
				}
			}
		}
	}()

	<-startSox
}
