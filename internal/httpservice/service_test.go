//go:build test && !integration

package httpservice

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Arten331/bot-checker/internal/config"
	"github.com/Arten331/bot-checker/internal/httpservice/httpwriter"
	"github.com/Arten331/observability/logger"
	"github.com/stretchr/testify/assert"
)

func TestHttpService_liveness(t *testing.T) {
	logger.MustSetupGlobal(
		logger.WithConfiguration(logger.CoreOptions{
			OutputPath: "stderr",
			Level:      logger.KeyLevelDebug,
			Encoding:   logger.EncodingConsole,
		}),
	)

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/liveness", nil)

	cfg, _ := config.Init()

	s, _ := New(
		WithHTTPAddress(net.JoinHostPort("", strconv.Itoa(cfg.HTTPService.Port))),
		WithResponseWritter(&httpwriter.JSONResponseWriter{}),
	)

	s.liveness().ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, 200)

	resp := &httpwriter.JSONResponseWithData{
		Message: "", // Must be OK
	}

	err := json.Unmarshal(rec.Body.Bytes(), resp)

	assert.NoError(t, err)

	if err != nil {
		return
	}

	assert.Equal(t, resp.Message, "OK")
}
