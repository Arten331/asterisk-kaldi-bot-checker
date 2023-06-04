package ari

import (
	"fmt"

	"github.com/Arten331/observability/logger"
	"github.com/CyCoreSystems/ari"
	"github.com/CyCoreSystems/ari/client/native"
	"github.com/inconshreveable/log15"
)

type Options struct {
	Host     string
	Port     int
	User     string
	Password string
	Original string
	Secure   bool
}

func New(o Options) ari.Client {
	wsProto := "ws"
	httpProto := "http"

	if o.Secure {
		wsProto = "wss"
		httpProto = "https"
	}

	url := fmt.Sprintf("%s://%s:%d/ari", httpProto, o.Host, o.Port)
	wsURL := fmt.Sprintf("%s://%s:%d/ari/events", wsProto, o.Host, o.Port)

	native.Logger = log15.New()
	native.Logger.SetHandler(loggerWrapper{})

	logger.L().Info("ari client build")

	cl := native.New(&native.Options{
		Application:     "bot_checker",
		URL:             url,
		WebsocketURL:    wsURL,
		WebsocketOrigin: o.Original,
		Username:        o.User,
		Password:        o.Password,
	})

	logger.L().Info("ari client connected")

	return cl
}

type loggerWrapper struct{}

func (l loggerWrapper) Log(r *log15.Record) error {
	switch r.Lvl {
	case log15.LvlCrit:
	case log15.LvlError:
		logger.L().Error(r.Msg)
	case log15.LvlWarn:
	case log15.LvlInfo:
		logger.L().Info(r.Msg)
	case log15.LvlDebug:
		logger.L().Debug(r.Msg)
	}

	return nil
}
