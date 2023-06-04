package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/Arten331/bot-checker/internal/models"
	"github.com/Arten331/observability/logger"
	"github.com/valyala/fastjson"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

const (
	BUFFSIZE = 8000
	EOF      = "{\"eof\" : 1}"
)

func main() {
	samplesPath := ""

	flag.StringVar(&samplesPath, "samplesDir", "", "")
	flag.Parse()

	dr, err := os.ReadDir(samplesPath)
	if err != nil {
		logger.L().Error("", zap.Error(err))

		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	newCsv := "samplesPath/csv_result/" + now + ".csv"

	csvFile, err := os.Create(newCsv)
	if err != nil {
		panic(err)
	}

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	for _, fInfo := range dr {
		if !fInfo.IsDir() {
			continue
		}

		categoryName := fInfo.Name()

		cDir, _ := os.ReadDir(samplesPath + "/" + categoryName)

		wg := sync.WaitGroup{}

		for _, recFile := range cDir {
			recordName := recFile.Name()
			recordPath := samplesPath + "/" + categoryName + "/" + recFile.Name()

			if recordName != "SUBSCRIBER_NOT_AVAIL.wav" {
				continue
			}

			wg.Add(1)

			go func() {
				checkAndWriteRecord(ctx, cancel, recordPath, recordName, categoryName, csvWriter)
				wg.Done()
			}()
		}

		wg.Wait()
	}
}

func checkAndWriteRecord(
	ctx context.Context,
	cancel context.CancelFunc,
	recordPath string,
	recordName string,
	categoryName string,
	csvWriter *csv.Writer,
) {
	var (
		audioReader io.Reader
		chCount     int
	)

	logger.L().Info("", zap.String("category", categoryName), zap.String("name", recordName))

	file, err := os.Open(recordPath)
	if err != nil {
		logger.L().Error("", zap.Error(err))

		return
	}

	chCount, err = getAudioChannels(recordPath)
	if err != nil {
		logger.L().Error("", zap.Error(err))

		return
	}

	if chCount != 1 && false {
		var newAudio *os.File

		newAudio, err = os.CreateTemp("/tmp", "*.wav")
		if err != nil {
			logger.L().Error("", zap.Error(err))

			return
		}

		_ = os.Remove(newAudio.Name())

		err = convertAudio(recordPath, newAudio.Name())
		if err != nil {
			logger.L().Error("", zap.Error(err))

			return
		}

		file, _ = os.Open(newAudio.Name())
	}

	audioReader = file

	resChan, errChan := processAudio(ctx, audioReader)

	for {
		select {
		case res := <-resChan:
			logger.L().Debug("read message", zap.Object("message", res))

			if res.IsFinal {
				logger.L().Info("End voice recognition", zap.Object("message", res))

				row := []string{string(res.Text), categoryName, recordName}

				err = csvWriter.Write(row)
				if err != nil {
					logger.L().Error("", zap.Error(err))
				}

				csvWriter.Flush()

				return
			}
		case err = <-errChan:
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logger.L().Info("stop recognition - canceled")
			} else {
				logger.L().Error("", zap.Error(err))
			}

			cancel()

			return
		}
	}
}

func processAudio(ctx context.Context, audio io.Reader) (resultChannel chan models.KaldiMessage, errChannel chan error) {
	resultChannel = make(chan models.KaldiMessage, 1)
	errChannel = make(chan error, 1)

	go func() {
		var cancel context.CancelFunc

		ctx, cancel = context.WithCancel(ctx)
		defer cancel()

		conn, _, err := websocket.Dial(ctx, "ws://localhost:2700", nil)
		if err != nil {
			errChannel <- err

			return
		}

		defer func() { _ = conn.Close(websocket.StatusInternalError, "unknown bullshit") }()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					buf := make([]byte, BUFFSIZE)
					dat, errRead := audio.Read(buf)

					if dat == 0 && errRead == io.EOF {
						err = conn.Write(ctx, websocket.MessageText, []byte(EOF))
						if err != nil {
							errChannel <- err

							return
						}

						return
					}

					err = conn.Write(ctx, websocket.MessageBinary, buf)
					if err != nil {
						errChannel <- err

						return
					}
				}
			}
		}()

		err = readMessages(ctx, conn, resultChannel)
		if err != nil {
			errChannel <- err
		}

		_ = conn.Close(websocket.StatusNormalClosure, "")

		return
	}()

	return resultChannel, errChannel
}

func readMessages(ctx context.Context, conn *websocket.Conn, ch chan<- models.KaldiMessage) error {
	var parser fastjson.Parser

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			_, msg, err := conn.Read(ctx)
			if err != nil {
				return err
			}

			v, err := parser.Parse(string(msg))
			if err != nil {
				return err
			}

			message := models.KaldiMessage{}

			value := v.Get("partial")
			if value != nil {
				message.Text, _ = value.StringBytes()
			} else {
				value = v.Get("text")
				message.Text, _ = value.StringBytes()
				message.IsFinal = true
			}

			ch <- message
		}
	}
}

func getAudioChannels(path string) (int, error) {
	ffprobe := exec.Command("/usr/local/bin/ffprobe", "-i", path, "-show_entries", "stream=channels:stream_tags=language", "-v", "0")

	regxpChannels := regexp.MustCompile(`channels=\d`)

	probeRes, err := ffprobe.Output()
	if err != nil {
		return 0, err
	}

	channelsRaw := regxpChannels.Find(probeRes)
	chSplit := bytes.Split(channelsRaw, []byte{'='})

	channelsNumber, err := strconv.Atoi(string(chSplit[1]))
	if err != nil {
		return 0, err
	}

	return channelsNumber, nil
}

func convertAudio(path, tmpNewPath string) error {
	ffmpeg := exec.Command("/usr/local/bin/ffmpeg", "-i", path, "-ac", "1", "-ar", "8000", tmpNewPath)

	output, err := ffmpeg.Output()
	if err != nil {
		logger.L().Debug("ffmpeg", zap.ByteString("output", output))

		return err
	}

	return nil
}
