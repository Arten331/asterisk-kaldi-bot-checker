package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Arten331/bot-checker/internal/app"
	"github.com/Arten331/bot-checker/internal/config"
	"github.com/Arten331/observability/logger"
)

func main() {
	var err error

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	cfg, err := config.Init()
	if err != nil {
		log.Panicf("Error configuration load, %v\n", err)
	}

	logger.MustSetupGlobal(
		logger.WithConfiguration(logger.CoreOptions{
			OutputPath: "stderr",
			Level:      cfg.Logger.Level,
			Encoding:   logger.EncodingConsole,
		}),
	)

	application, err := app.Init(ctx, &cfg)
	if err != nil {
		log.Panicf("Error load application modulues, %v\n", err)
	}

	err = application.Run(ctx, stop)
	if err != nil {
		log.Panicf("Error run application %v\n", err)
	}

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig

		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()

			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Panicf("graceful shutdown timed out.. forcing exit.")
			}
		}()

		err := application.Shutdown(ctx)
		if err != nil {
			log.Fatalln(err)
		}

		stop()
		cancel()
	}()

	<-ctx.Done()

	<-time.After(time.Second * 10)

	logger.L().Info("shutdown timed out.. forcing exit.")
}
