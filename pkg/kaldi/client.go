package kaldi

import (
	"context"
	"fmt"
	"io"

	"github.com/Arten331/bot-checker/internal/models"
	"github.com/Arten331/bot-checker/pkg/audio"
	"github.com/Arten331/observability/logger"
	"github.com/valyala/fastjson"
	"nhooyr.io/websocket"
)

const (
	BUFFSIZE = audio.BUFFSIZE
	EOF      = "{\"eof\" : 1}"
)

type Options struct {
	Host string
	Port int
}

type Client struct {
	KaldiURL string
}

func NewClient(o Options) *Client {
	c := &Client{
		KaldiURL: fmt.Sprintf("ws://%s:%d/", o.Host, o.Port),
	}

	return c
}

func (c *Client) ProcessAudio(ctx context.Context, reader io.Reader) (resultChannel chan models.KaldiMessage, errChannel chan error) {
	resultChannel = make(chan models.KaldiMessage, 1)
	errChannel = make(chan error, 1)

	go func() {
		var cancel context.CancelFunc

		ctx, cancel = context.WithCancel(ctx)
		defer cancel()

		conn, _, err := websocket.Dial(ctx, c.KaldiURL, nil)
		if err != nil {
			errChannel <- err

			return
		}

		defer func() { _ = conn.Close(websocket.StatusInternalError, "oops, unknown problem") }()

		go func() {
			for {
				select {
				case <-ctx.Done():
					logger.L().Debug("ctx done, stop kaldi process")

					return
				default:
					buf := make([]byte, BUFFSIZE)
					dat, errRead := reader.Read(buf)

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

		err = c.readMessages(ctx, conn, resultChannel)
		if err != nil {
			errChannel <- err
		}

		_ = conn.Close(websocket.StatusNormalClosure, "")

		return
	}()

	return resultChannel, errChannel
}

func (c *Client) readMessages(ctx context.Context, conn *websocket.Conn, ch chan<- models.KaldiMessage) error {
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

				ch <- message

				continue
			}

			ch <- message
		}
	}
}
