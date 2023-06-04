//go:build test && !integration

package audio_test

import (
	"context"
	"io"
	"io/fs"
	"os"
	"testing"
	"time"

	testdata "github.com/Arten331/bot-checker/data/test"
	"github.com/Arten331/bot-checker/pkg/audio"
	commands2 "github.com/Arten331/bot-checker/pkg/audio/commands"
	"github.com/stretchr/testify/require"
)

func TestPipe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fileReaded := make(chan interface{}, 1)

	testFs := testdata.GetTestFS()
	fileTest, err := testFs.Open("sox/raw_from_fork.pcm")
	require.NoError(t, err)

	pipe, err := audio.NewPipe([]audio.PipeCommand{
		commands2.NewPcmRawToWav(),
		commands2.NewPcmWavSilence(),
	})
	require.NoError(t, err)

	err = os.MkdirAll("/tmp/test", fs.ModeDir|0777)

	require.NoError(t, err)

	resultAudio, err := os.Create("/tmp/test/pipe" + time.Now().Format(time.RFC3339Nano) + ".wav")
	require.NoError(t, err)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				buf := make([]byte, audio.BUFFSIZE)
				_, err := fileTest.Read(buf)
				if err == io.EOF {
					fileReaded <- nil

					return
				}

				err = pipe.Write(buf)

				require.NoError(t, err)
			}
		}
	}()

	go func() {
		err := <-pipe.StdErr

		t.Errorf("Error: %v", err)
		t.Failed()

		return
	}()

	err = pipe.Run(ctx)
	require.NoError(t, err)

	go func() {
		for {
			buf := make([]byte, audio.BUFFSIZE)

			_, err = pipe.StdOut.Read(buf)
			if err == io.EOF {
				break
			}
			require.NoError(t, err)

			_, err = resultAudio.Write(buf)
			require.NoError(t, err)
		}
	}()

	select {
	case <-fileReaded:
		<-time.After(2 * time.Second)
		t.Log("file reader, 2 seconds ago")
	case <-ctx.Done():
		t.Log("ctx done")
	}

	statResAudio, _ := resultAudio.Stat()
	sizeResAudio := statResAudio.Size()

	expectedFile, err := testFs.Open("sox/pipe_silenced.wav")
	require.NoError(t, err)

	//compare results
	statExpected, _ := expectedFile.Stat()
	sizeExpected := statExpected.Size()

	require.Equalf(t, sizeExpected, sizeResAudio, "not expected file: %s", statResAudio.Name())
}
