package app

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/Arten331/bot-checker/internal/agiservice"
	"github.com/Arten331/bot-checker/internal/app/global"
	"github.com/Arten331/bot-checker/internal/botchecker"
	"github.com/Arten331/bot-checker/internal/config"
	"github.com/Arten331/bot-checker/internal/domain/phrase"
	"github.com/Arten331/bot-checker/internal/domain/phrase/memdb"
	"github.com/Arten331/bot-checker/internal/events"
	checkevents "github.com/Arten331/bot-checker/internal/events/bot_checker"
	"github.com/Arten331/bot-checker/internal/httpservice"
	"github.com/Arten331/bot-checker/internal/httpservice/httpwriter"
	"github.com/Arten331/bot-checker/pkg/ari"
	"github.com/Arten331/bot-checker/pkg/kaldi"
	kafkaClient "github.com/Arten331/messaging/kafka"
	"github.com/Arten331/observability/metrics"
)

type Repositories struct {
	stopPhrases phrase.Repository
}

type Services struct {
	httpService *httpservice.Service
	agiService  *agiservice.Service
	botChecker  *botchecker.BotChecker
}

type App struct {
	serviceName  string
	env          string
	cfg          *config.AppConfig
	services     Services
	metrics      *metrics.Service
	repositories Repositories
	events       struct {
		publisher events.EventPublisher
	}
}

func Init(ctx context.Context, cfg *config.AppConfig) (ac *App, err error) {
	ms := metrics.New()

	ac = &App{
		serviceName: cfg.App.Name,
		env:         cfg.App.Env,
		cfg:         cfg,
		metrics:     &ms,
	}

	err = ac.initRepositories(ctx)
	if err != nil {
		return nil, err
	}

	ac.initEventsServices(ctx)

	err = ac.initServices(ctx)
	if err != nil {
		return nil, err
	}

	// setup globals
	global.SetGlobals(ac.serviceName, ac.env)

	return ac, nil
}

func (a *App) initServices(_ context.Context) error {
	kaldiClient := kaldi.NewClient(kaldi.Options{
		Host: a.cfg.Kaldi.Host,
		Port: a.cfg.Kaldi.Port,
	})

	ariCfg := a.cfg.Ari
	ariClient := ari.New(ari.Options{
		Host:     ariCfg.Host,
		Port:     ariCfg.Port,
		User:     ariCfg.User,
		Password: ariCfg.Password,
		Original: ariCfg.Original,
		Secure:   ariCfg.Secure,
	})

	botCheckService, err := botchecker.New(&botchecker.Options{
		MetricService:         a.metrics,
		StopPhrasesRepository: a.repositories.stopPhrases,
		KaldiClient:           kaldiClient,
		AriClient:             ariClient,
		EventPublisher:        a.events.publisher,
	})
	if err != nil {
		return err
	}

	agiService := agiservice.New(agiservice.Options{
		Host:    a.cfg.Agi.Host,
		Port:    a.cfg.Agi.Port,
		Handler: botCheckService,
	})

	rw := httpwriter.NewJSONResponseWriter()

	httpService, err := httpservice.New(
		httpservice.WithHTTPAddress(net.JoinHostPort("", strconv.Itoa(a.cfg.HTTPService.Port))),
		httpservice.WithResponseWritter(&rw),
		httpservice.WithServices(httpservice.Services{
			Metrics:    a.metrics,
			BotChecker: botCheckService,
		}),
	)
	if err != nil {
		return err
	}

	a.services = Services{
		httpService: httpService,
		agiService:  agiService,
		botChecker:  botCheckService,
	}

	return err
}

func (a *App) initRepositories(_ context.Context) error {
	stopPhraseRepo, err := memdb.NewPhraseMemDBRepository()
	if err != nil {
		return err
	}

	a.repositories.stopPhrases = &stopPhraseRepo

	return nil
}

func (a *App) initEventsServices(_ context.Context) {
	brokers := make([]string, 0, len(a.cfg.QueueService.Kafka.BootstrapServers))

	for _, server := range a.cfg.QueueService.Kafka.BootstrapServers {
		brokers = append(brokers, fmt.Sprintf("%s:%d", server, a.cfg.QueueService.Kafka.Port))
	}

	cfgQueue := a.cfg.QueueService

	a.events.publisher = events.NewEventPublisher()

	eventClickStat := events.NewKafkaEventHandler(kafkaClient.MustCreateProducer(kafkaClient.ProducerClientOptions{
		Brokers: brokers,
		Topic:   cfgQueue.Topics.ClickData.Name,
	}))

	// публикуем события
	a.events.publisher.Subscribe(eventClickStat,
		&checkevents.BotFound{},
		&checkevents.BotNotFounded{},
	)
}

func (a *App) Run(ctx context.Context, cancelFunc context.CancelFunc) error {
	go a.services.httpService.Run(ctx, cancelFunc)
	go a.services.agiService.Run(ctx, cancelFunc)
	go a.services.botChecker.Run(ctx, cancelFunc)

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	var err error

	err = a.services.httpService.Shutdown(ctx)
	if err != nil {
		return err
	}

	err = a.services.agiService.Shutdown(ctx)
	if err != nil {
		return err
	}

	return err
}
