package httpservice

import (
	"net/http"

	"github.com/Arten331/observability/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Service) liveness() http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		s.writer.WriteSuccess(w, "OK", nil)
	}
}

func (s *Service) readiness() http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		s.writer.WriteSuccess(w, "OK", nil)
	}
}

func (s *Service) prometheus() http.HandlerFunc {
	if s.services.Metrics != nil {
		logger.L().Info("custom prometheus registry handler enabled")

		return s.services.Metrics.Handler()
	}

	logger.L().Info("default prometheus registry enabled")

	return func(w http.ResponseWriter, r *http.Request) {
		promhttp.HandlerFor(
			prometheus.DefaultGatherer,
			promhttp.HandlerOpts{},
		).ServeHTTP(w, r)
	}
}
