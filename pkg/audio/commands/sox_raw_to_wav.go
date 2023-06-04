package commands

import (
	"bytes"
	"context"
	"io"
	"os/exec"

	"github.com/Arten331/bot-checker/pkg/audio"
	"github.com/Arten331/observability/logger"
	"go.uber.org/zap"
)

type PcmRawToWav struct{}

func NewPcmRawToWav() *PcmRawToWav {
	return &PcmRawToWav{}
}

func (p *PcmRawToWav) Name() string {
	return "sox_raw_to_wav"
}

func (p *PcmRawToWav) Handle(ctx context.Context, errChan audio.ErrChan) (in io.Writer, out io.Reader) {
	ctx, cancel := context.WithCancel(ctx)

	in = bytes.NewBuffer([]byte{})
	out = bytes.NewBuffer([]byte{})

	com := exec.CommandContext(ctx, "sox",
		"-v", "1", "--ignore-length", "--buffer", "8000", "-t", "raw", "-r", "8000", "-e", "signed", "-c", "1",
		"-b", "16", "-", "-t", "wav", "-", "-q")

	stdOut, _ := com.StdoutPipe()
	stdIn, _ := com.StdinPipe()
	stdErr, _ := com.StderrPipe()

	err := com.Start()

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
