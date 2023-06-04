package httpservice

import (
	"net/http"
	"time"

	"github.com/Arten331/bot-checker/internal/httpservice/mwwrapper"
	"github.com/Arten331/observability/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	KeyGroupBase mwwrapper.MiddlewareGroupKey = iota + 1
)

func (s *Service) configureMiddlewares() {
	groups := mwwrapper.MiddlewareGroups{
		KeyGroupBase: &chi.Middlewares{
			ZapLogger,
			middleware.Recoverer,
		},
	}

	mw := mwwrapper.NewMiddleWareWrapper(mwwrapper.Options{
		Groups: &groups,
	})

	s.middlewares = mw
}

func ZapLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		latency := time.Since(start)

		fields := []zapcore.Field{
			zap.Int("status", ww.Status()),
			zap.Duration("took", latency),
			zap.String("remote", r.RemoteAddr),
			zap.String("request", r.RequestURI),
			zap.String("method", r.Method),
		}

		logger.L().Info("request completed", fields...)
	}

	return http.HandlerFunc(fn)
}
