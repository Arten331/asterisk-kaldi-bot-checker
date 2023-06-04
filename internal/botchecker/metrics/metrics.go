package metrics

import (
	"github.com/Arten331/bot-checker/internal/domain/phrase"
	"github.com/Arten331/observability/logger"
	"github.com/Arten331/observability/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const label = "asterisk"

type Metrics struct {
	Service    metrics.PrometheusService
	collectors MetricCollectors
}

type MetricCollectors struct {
	waitForNoiseHangup *prometheus.CounterVec
	ivrCheckStart      *prometheus.CounterVec
	ivrCheckHangup     *prometheus.CounterVec
}

type WaitForNoise struct {
	Phase    string
	Caller   string
	Dnid     string
	Result   string
	Campaign string
}

func (m *Metrics) Collectors() MetricCollectors {
	return m.collectors
}

func (m *Metrics) Register() {
	waitForNoiseHangup := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wait_for_noise_hangup",
			Help: "Wait for noise count hangup/queued",
		},
		[]string{"phase", "result", "campaign", "group"},
	)

	ivrCheckStart := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ivr_check_start",
			Help: "Wait for noise count hangup/queued",
		},
		[]string{"group"},
	)

	ivrCheckHangup := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ivr_check_hangup",
			Help: "Wait for noise count hangup/queued",
		},
		[]string{"phrase", "group"},
	)

	m.collectors = MetricCollectors{
		waitForNoiseHangup: waitForNoiseHangup,
		ivrCheckStart:      ivrCheckStart,
		ivrCheckHangup:     ivrCheckHangup,
	}

	_ = m.Service.Register(waitForNoiseHangup)
	_ = m.Service.Register(ivrCheckStart)
	_ = m.Service.Register(ivrCheckHangup)
}

func (m *Metrics) StoreNoiseHangup(r *WaitForNoise) {
	m.collectors.waitForNoiseHangup.WithLabelValues(r.Phase, r.Result, r.Campaign, label).Inc()
	logger.L().Debug("stored wait for noise", zap.Object("result", r))
}

func (m *Metrics) StoreIvrCheckStart() {
	m.collectors.ivrCheckStart.WithLabelValues(label).Inc()
	logger.L().Debug("stored ivr check start")
}

func (m *Metrics) StoreIvrCheckHangup(p *phrase.StopPhrase) {
	m.collectors.ivrCheckHangup.WithLabelValues(p.Phrase, label).Inc()
	logger.L().Debug("stored ivr check hangup")
}

func (m *Metrics) ResetNoiseHangup() {
	m.collectors.waitForNoiseHangup.Reset()
}

func (w *WaitForNoise) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("phase", w.Phase)
	enc.AddString("caller", w.Caller)
	enc.AddString("dnid", w.Dnid)
	enc.AddString("result", w.Result)
	enc.AddString("campaign", w.Campaign)

	return nil
}
