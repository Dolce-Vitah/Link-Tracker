package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/botclient"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/external"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/grpcscrapper"
	httpscrapper "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/scrapper"
	"google.golang.org/grpc"
)

type App struct {
	config     *config.Config
	service    *repository.Service
	httpServer *httpscrapper.HTTPServer
	scheduler  *scrapper.Scheduler
	interval   time.Duration
}

func (a *App) New() error {
	cfg, err := config.Load("config.json")
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
	httpServer := httpscrapper.NewHTTPServer(service)
	externalClient := external.NewHTTPClient(timeout)
	updatesClient := botclient.NewHTTPUpdatesClient(cfg.BotBaseURL, timeout)
	scheduler := scrapper.NewScheduler(service, externalClient, updatesClient, interval)

	a.config = cfg
	a.service = service
	a.httpServer = httpServer
	a.scheduler = scheduler
	a.interval = interval

	return nil
}

func (a *App) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go a.runScheduledChecks(ctx)
	go a.runGRPCServer()

	slog.Info("Scrapper HTTP server is up", slog.String("address", a.config.ScrapperHTTPAddress))
	if err := http.ListenAndServe(a.config.ScrapperHTTPAddress, a.httpServer.Handler()); err != nil {
		slog.Error("Scrapper HTTP server failed", slog.String("error", err.Error()))
		os.Exit(1)
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

func (a *App) runGRPCServer() {
	listener, err := net.Listen("tcp", a.config.ScrapperGRPCAddress)
	if err != nil {
		slog.Error("Failed to listen grpc",
			slog.String("address", a.config.ScrapperGRPCAddress),
			slog.String("error", err.Error()),
		)
		return
	}

	grpcServer := grpc.NewServer()
	grpcscrapper.Register(grpcServer, a.service)

	slog.Info("Scrapper gRPC server is up", slog.String("address", a.config.ScrapperGRPCAddress))
	if err := grpcServer.Serve(listener); err != nil {
		slog.Error("Scrapper gRPC server failed", slog.String("error", err.Error()))
	}
}
