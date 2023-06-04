package commands

import (
	"bytes"
	"context"
	"io"
	"os/exec"

	"github.com/Arten331/bot-checker/pkg/audio"
	"github.com/Arten331/observability/logger"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

type PcmWavStripSilence struct{}

func NewPcmWavSilence() *PcmWavStripSilence {
	return &PcmWavStripSilence{}
}

func (p *PcmWavStripSilence) Name() string {
	return "sox wav strip silence"
}

func (p *PcmWavStripSilence) Handle(ctx context.Context, errChan audio.ErrChan) (in io.Writer, out io.Reader) {
	ctx, cancel := context.WithCancel(ctx)

	in = bytes.NewBuffer([]byte{})
	out = bytes.NewBuffer([]byte{})

	com := exec.CommandContext(ctx, "sox",
		"-v", "1", "--ignore-length", "--buffer", "8000", "-q", "-t", "wav", "-c", "1", "-",
		"-t", "wav", "-", "silence", "-l", "1", "0.1", "1%", "-1", "2.0", "1%")

	stdOut, _ := com.StdoutPipe()
	stdIn, _ := com.StdinPipe()
	stdErr, _ := com.StderrPipe()

	err := com.Start()

	rlErrorCheck := ratelimit.New(1000) // per second

	if err != nil {
		errChan <- audio.NewErr(p, err.Error())

		cancel()

		return in, out
	}

	go func() {
		_ = com.Wait()

		cancel()
	}()

	go func() {
		for {
			rlErrorCheck.Take()

			select {
			case <-ctx.Done():
				return
			default:
				errorBuf, _ := io.ReadAll(stdErr)
				if len(errorBuf) != 0 {
					if bytes.Contains(errorBuf, []byte("Length in output .wav header will be wrong")) {
						continue
					}

					logger.L().Info("sox strip silence error", zap.ByteString("err", errorBuf))
					errChan <- audio.NewErr(p, string(errorBuf))
				}
			}
		}
	}()

	return stdIn, stdOut
}
