package httpservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/Arten331/bot-checker/internal/botchecker"
	"github.com/Arten331/bot-checker/internal/httpservice/httpwriter"
	"github.com/Arten331/bot-checker/internal/httpservice/mwwrapper"
	"github.com/Arten331/observability/logger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type MetricsService interface {
	Handler() http.HandlerFunc
}

type Services struct {
	Metrics    MetricsService
	BotChecker *botchecker.BotChecker
}

type Service struct {
	server      *http.Server
	middlewares mwwrapper.MiddlewareWrapper
	router      *chi.Mux
	writer      httpwriter.Writer
	services    Services
	context     context.Context
}

type Configuration func(s *Service) error

func New(cfgs ...Configuration) (*Service, error) {
	service := Service{}

	// Apply all Configurations passed in
	for _, cfg := range cfgs {
		err := cfg(&service)
		if err != nil {
			return nil, err
		}
	}

	service.configureMiddlewares()
	service.configureRouter()

	return &service, nil
}

func (s *Service) Run(ctx context.Context, cancel context.CancelFunc) {
	s.context = ctx

	go func() {
		logger.L().Info(fmt.Sprintf("Start http httpserver on %s!", s.server.Addr))

		err := s.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.L().Error("error serve httpserver", zap.Error(err))
			cancel()
		}
	}()

	<-ctx.Done()

	logger.L().Info(fmt.Sprintf("Shutdown http httpserver on %s!", s.server.Addr))

	err := s.server.Shutdown(context.Background())
	if err != nil {
		logger.L().Error("Unable shutdown http httpserver", zap.Error(err))
	}
}

func (s *Service) Shutdown(shutdownCtx context.Context) error {
	return s.server.Shutdown(shutdownCtx)
}

func (s *Service) configureRouter() {
	mwGroups := s.middlewares.Groups

	s.enablePPROFHandlers()
	s.router.With(mwGroups.GetChain(KeyGroupBase)...).Route("/", func(r chi.Router) {
		r.Get("/metrics", s.prometheus())
	})

	s.router.Get("/readiness", s.readiness())
	s.router.Get("/liveness", s.liveness())

	s.router.With(mwGroups.GetChain(KeyGroupBase)...).Handle("/bot-check/{uniqID}", s.services.BotChecker.CheckBotHandler())
}

func WithHTTPAddress(address string) Configuration {
	return func(s *Service) error {
		s.router = chi.NewRouter()
		s.server = &http.Server{
			Addr:    address,
			Handler: s.router,
		}

		return nil
	}
}

func WithServices(services Services) Configuration {
	return func(s *Service) error {
		s.services = services

		return nil
	}
}

func WithResponseWritter(r httpwriter.Writer) Configuration {
	return func(s *Service) error {
		s.writer = r

		return nil
	}
}

func (s *Service) enablePPROFHandlers() {
	s.router.With(s.middlewares.Groups.GetChain(KeyGroupBase)...).Route("/debug", func(r chi.Router) {
		r.HandleFunc("/pprof", pprof.Index)
		r.HandleFunc("/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/pprof/profile", pprof.Profile)
		r.HandleFunc("/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/pprof/trace", pprof.Trace)
		r.Handle("/pprof/goroutine", pprof.Handler("goroutine"))
		r.Handle("/pprof/threadcreate", pprof.Handler("threadcreate"))
		r.Handle("/pprof/mutex", pprof.Handler("mutex"))
		r.Handle("/pprof/heap", pprof.Handler("heap"))
		r.Handle("/pprof/block", pprof.Handler("block"))
		r.Handle("/pprof/allocs", pprof.Handler("allocs"))
	})
}
