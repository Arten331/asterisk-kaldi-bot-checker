//go:build test && !integration

package metrics_test

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"testing"

	"github.com/Arten331/bot-checker/internal/botchecker/metrics"
	"github.com/Arten331/bot-checker/internal/httpservice"
	"github.com/Arten331/bot-checker/internal/httpservice/httpwriter"
	metricsService "github.com/Arten331/bot-checker/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPort = ":8088"

func TestBotcheckerMetrics(t *testing.T) {
	ms := metricsService.New()

	rw := httpwriter.NewJSONResponseWriter()
	httpService, err := httpservice.New(
		httpservice.WithHTTPAddress(testPort),
		httpservice.WithResponseWritter(&rw),
		httpservice.WithServices(httpservice.Services{
			Metrics: &ms,
		}),
	)
	assert.NoError(t, err)

	botcheckerMetrics := metrics.Metrics{
		Service: &ms,
	}
	botcheckerMetrics.Register()

	randomPhases := [...]string{"call_record", "hello", "queue", "hearme"}
	randomCaller := [...]string{"979144181775", "979144181774", "979144181773", "979144181772"}
	randomDest := [...]string{"88005003333", "88005004444", "88005005555", "88005006666"}
	randomRes := [...]string{"HANGUP", "QUEUE"}

	go httpService.Run(context.TODO(), func() {})

	wfnCount := 10

	wfnStored := make([]metrics.WaitForNoise, 0, wfnCount)

	for i := 0; i < wfnCount; i++ {
		randomPhase := randomPhases[rand.Intn(4)]
		wfn := &metrics.WaitForNoise{
			Phase:  randomPhase,
			Caller: randomCaller[rand.Intn(4)],
			Dnid:   randomDest[rand.Intn(4)],
			Result: randomRes[rand.Intn(2)],
		}

		wfnStored = append(wfnStored, *wfn)

		botcheckerMetrics.StoreNoiseHangup(wfn)
	}

	get, err := http.Get("http://localhost" + testPort + "/metrics")
	require.NoError(t, err)

	body, err := io.ReadAll(get.Body)
	require.NoError(t, err)

	//find wait for stored
	for _, stored := range wfnStored {
		find := regexp.MustCompile(fmt.Sprintf("wait_for_noise_hangup{campaign=\"\",group=\"asterisk\",phase=\"%s\",result=\"%s\"} \\d", stored.Phase, stored.Result))

		assert.Regexp(t, body, find)
	}
}
