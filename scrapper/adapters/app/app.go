package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-co-op/gocron"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/scheduler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/botclient"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/external"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/grpcserver"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/httpserver"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"google.golang.org/grpc"
)

type App struct {
	config      *config.Config
	service     *repository.Service
	httpHandler *httpserver.Handler
	scheduler   *scheduler.Scheduler
	interval    time.Duration
}

func (a *App) New() error {
	cfg, err := config.Load("config.scrapper.json")
	if err != nil {
		return fmt.Errorf("load app config: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.ExternalHTTPTimeout)
	if err != nil {
		return fmt.Errorf("parse external http timeout: %w", err)
	}

	interval, err := time.ParseDuration(cfg.SchedulerInterval)
	if err != nil {
		return fmt.Errorf("parse scheduler interval: %w", err)
	}

	service := repository.NewService()
	httpHandler := httpserver.NewHandler(service)
	externalClient := external.NewHTTPClient(timeout)
	updatesClient := botclient.NewHTTPUpdatesClient(cfg.BotBaseURL, timeout)

	scheduler := scheduler.New(service, externalClient, updatesClient, interval)

	a.config = cfg
	a.service = service
	a.httpHandler = httpHandler
	a.scheduler = scheduler
	a.interval = interval

	return nil
}

func (a *App) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go a.runScheduledChecks(ctx)

	go a.runGRPCServer(ctx)

	slog.Info("Scrapper HTTP server is up", slog.String("address", a.config.ScrapperHTTPAddress))
	if err := http.ListenAndServe(a.config.ScrapperHTTPAddress, a.httpHandler.Handler()); err != nil {
		slog.Error("Scrapper HTTP server failed", slog.String("error", err.Error()))
		return
	}
}

func (a *App) runScheduledChecks(ctx context.Context) {
	cron := gocron.NewScheduler(time.UTC)
	_, err := cron.Every(a.interval).Do(func() {
		a.scheduler.ProcessOnce(ctx)
	})
	if err != nil {
		slog.Error("Failed to schedule link checks", slog.String("error", err.Error()))
		return
	}

	cron.StartAsync()

	<-ctx.Done()

	cron.Stop()
}

func (a *App) runGRPCServer(ctx context.Context) {
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "tcp", a.config.ScrapperGRPCAddress)
	if err != nil {
		slog.Error("Failed to listen grpc",
			slog.String("address", a.config.ScrapperGRPCAddress),
			slog.String("error", err.Error()),
		)
		return
	}

	grpcServer := grpc.NewServer()
	grpcAppServer := grpcserver.NewServer(a.service)

	grpcAppServer.Register(grpcServer)

	slog.Info("Scrapper gRPC server is up", slog.String("address", a.config.ScrapperGRPCAddress))
	if serveErr := grpcServer.Serve(listener); serveErr != nil {
		slog.Error("Scrapper gRPC server failed", slog.String("error", serveErr.Error()))
	}
}
