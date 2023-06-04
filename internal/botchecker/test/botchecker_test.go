//go:build test && integration

package botchecker_test

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"testing"

	testdata "github.com/Arten331/bot-checker/data/test"
	"github.com/Arten331/bot-checker/internal/botchecker"
	"github.com/Arten331/bot-checker/internal/config"
	"github.com/Arten331/bot-checker/internal/domain/phrase"
	"github.com/Arten331/bot-checker/internal/domain/phrase/memdb"
	"github.com/Arten331/bot-checker/internal/httpservice"
	"github.com/Arten331/bot-checker/internal/httpservice/httpwriter"
	"github.com/Arten331/bot-checker/pkg/kaldi"
	"github.com/Arten331/bot-checker/pkg/metrics"
	"github.com/stretchr/testify/require"
)

type FileTestCase struct {
	record   string
	expected *phrase.StopPhrase
}

var (
	servicesStarted bool
	testFS          *embed.FS
	checker         *botchecker.BotChecker
	kaldiClient     *kaldi.Client
	httpService     *httpservice.Service
	testHost        = ":8083"
)

func initServices(t *testing.T) {
	if servicesStarted {
		return
	}

	cfg, err := config.Init()
	require.NoError(t, err)

	testFS = testdata.GetTestFS()

	kaldiClient = kaldi.NewClient(kaldi.Options{
		Host: cfg.Kaldi.Host,
		Port: cfg.Kaldi.Port,
	})

	ms := metrics.New()

	stopPhraseRepo, err := memdb.NewPhraseMemDBRepository()
	require.NoError(t, err)

	checker, err = botchecker.New(&botchecker.Options{
		MetricService:         &ms,
		StopPhrasesRepository: &stopPhraseRepo,
		KaldiClient:           kaldiClient,
	})
	require.NoError(t, err)

	go checker.Run(context.Background(), func() {})

	rw := httpwriter.NewJSONResponseWriter()
	httpService, err = httpservice.New(
		httpservice.WithHTTPAddress(testHost),
		httpservice.WithResponseWritter(&rw),
		httpservice.WithServices(httpservice.Services{
			Metrics: &ms,
		}),
	)
	require.NoError(t, err)

	go httpService.Run(context.Background(), func() {})
}

func TestBotChecker_Check(t *testing.T) {
	initServices(t)

	testCases := []FileTestCase{
		{
			record: "20220216_SUBSCRIBER_NOT_AVAIL.wav",
			expected: &phrase.StopPhrase{
				Phrase:   "абонент временно недоступен",
				Category: phrase.Category("unavilable"),
			},
		},
		{
			record: "20220217_BLOCKED.wav",
			expected: &phrase.StopPhrase{
				Phrase:   "извините набранный",
				Category: phrase.Category("new"),
			},
		},
		{
			record: "20220217_BUSY_WAITING.wav",
			expected: &phrase.StopPhrase{
				Phrase:   "пожалуйста оставайтесь",
				Category: phrase.Category("busy_waiting"),
			},
		},
		{
			record: "20220217_DISCONNECTED1.wav",
			expected: &phrase.StopPhrase{
				Phrase:   "телефон абонента выключен",
				Category: phrase.Category("disconnected"),
			},
		},
		{
			record: "ME_ABONENT_NE_MOJZHET.wav",
			expected: &phrase.StopPhrase{
				Phrase:   "абонент не может ответить",
				Category: phrase.Category("new"),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.record, func(t *testing.T) {
			file, err := testFS.Open("records/" + testCase.record)
			defer func() { _ = file.Close() }()
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			resCh, errCh := kaldiClient.ProcessAudio(ctx, file)

			_, stopPhrase, err := checker.Check(ctx, cancel, resCh, errCh)
			require.EqualValues(t, testCase.expected, stopPhrase)
		})
	}

	t.Run("check works metrics", testMetrics)
}

func testMetrics(t *testing.T) {
	get, err := http.Get("http://localhost" + testHost + "/metrics")
	require.NoError(t, err)

	fmt.Println(get)
}
