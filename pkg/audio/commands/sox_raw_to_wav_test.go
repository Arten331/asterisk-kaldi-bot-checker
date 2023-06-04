//go:build test && !integration

package commands

import (
	"context"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	testdata "github.com/Arten331/bot-checker/data/test"
	"github.com/Arten331/bot-checker/pkg/audio"
	"github.com/stretchr/testify/require"
	"go.uber.org/ratelimit"
)

func TestPcmRawToWav(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	soxHandler := NewPcmRawToWav()

	errChan := make(audio.ErrChan, 1)

	in, out := soxHandler.Handle(ctx, errChan)

	go func() {
		err := <-errChan

		t.Errorf("Error: %v", err)
		return
	}()

	testFS := testdata.GetTestFS()
	testFile, err := testFS.Open("sox/raw_from_fork.pcm")
	require.NoError(t, err)

	pcmReaden := make(chan interface{})

	go func() {
		rl := ratelimit.New(1000)
		for {
			rl.Take()
			rms := rand.Intn(100)
			time.Sleep(time.Microsecond * time.Duration(rms))

			buf := make([]byte, 320)
			_, err := testFile.Read(buf)
			if err == io.EOF {
				pcmReaden <- nil

				return
			}
			require.NoError(t, err)

			_, err = in.Write(buf)
			require.NoError(t, err)
		}
	}()

	var newAudio *os.File

	newAudio, err = os.Create("/tmp/wav_" + time.Now().Format(time.RFC3339Nano) + ".wav")
	require.NoError(t, err)

	go func() {
		_, err = io.Copy(newAudio, out)
		require.NoError(t, err)
	}()

	<-pcmReaden

	time.Sleep(1000 * time.Millisecond)

	expectedFile, err := testFS.Open("sox/raw_from_fork_wav.wav")
	require.NoError(t, err)

	//compare results
	statExpected, _ := expectedFile.Stat()
	statNew, _ := newAudio.Stat()

	require.Conditionf(t, func() (success bool) {
		return statNew.Size() > statExpected.Size()-statExpected.Size()*10/100
	}, "Not equal size expected wav - created")
}
