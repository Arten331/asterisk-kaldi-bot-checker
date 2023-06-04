package agiservice

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/Arten331/observability/logger"
	"github.com/zaf/agi"
	"go.uber.org/zap"
)

type AgiSessionHandler interface {
	AgiHandler(session *agi.Session) error
}

type Service struct {
	address  string
	listener net.Listener
	handler  AgiSessionHandler
}

type Options struct {
	Host    string
	Port    int
	Handler AgiSessionHandler
}

func New(o Options) *Service {
	address := net.JoinHostPort(o.Host, strconv.Itoa(o.Port))

	return &Service{
		address: address,
		handler: o.Handler,
	}
}

func (s *Service) Run(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	var err error

	s.listener, err = net.Listen("tcp", s.address)
	if err != nil {
		logger.L().Error("AGI listener error", zap.Error(err))

		return
	}

	logger.L().Info(fmt.Sprintf("AGI service listen on %s", s.address))

	for {
		select {
		case <-ctx.Done():
			err = s.listener.Close()
			if err != nil {
				logger.S().Infof("Unable stop agi service: %s", err.Error())
			}

			logger.S().Infof("Stopped agi service %s", s.listener.Addr())

			return
		default:
			conn, err := s.listener.Accept()
			if err != nil || conn == nil {
				logger.L().Error("AGI accept connection error", zap.Error(err))

				continue
			}

			go func() {
				err = s.handleAgi(conn)

				if err != nil {
					logger.L().Error("Error handle AGI connect", zap.Error(err))
				}
			}()
		}
	}
}

func (s *Service) Shutdown(_ context.Context) error {
	return s.listener.Close()
}

func (s *Service) handleAgi(c net.Conn) error {
	var (
		err        error
		rw         *bufio.ReadWriter
		agiSession = agi.New()
	)

	defer func() {
		if err := recover(); err != nil {
			logger.L().Error("Session terminated:", zap.Any("error", err))
		}

		_ = c.Close()
	}()

	if c != nil {
		rw = bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))
	}

	logger.L().Debug("handle AGI", zap.String("remote", c.RemoteAddr().String()))

	err = agiSession.Init(rw)
	if err != nil {
		logger.L().Error("Error Parsing AGI environment: %v\n", zap.Error(err))

		return err
	}

	return s.handler.AgiHandler(agiSession)
}
