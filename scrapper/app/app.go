package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/poller"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/botclient"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/db"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/db/migrations"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/external"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/grpcserver"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/httpserver"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository/ormrepo"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository/sqlrepo"
	"google.golang.org/grpc"
)

type App struct {
	config      *config.Config
	service     repository.Service
	httpHandler *httpserver.Handler
	poller      *poller.Poller
	interval    time.Duration
	outbox      *poller.OutboxDispatcher
	outboxTick  time.Duration
	sqlDB       *sql.DB
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

	connMaxLifetime, lifeErr := time.ParseDuration(cfg.DBConnMaxLifetime)
	if lifeErr != nil {
		return fmt.Errorf("parse db conn max lifetime: %w", lifeErr)
	}
	dbOpts := db.Options{
		DSN:             cfg.DBDsn,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
	}
	service, sqlDB, serviceErr := createRepositoryByAccessType(cfg.AccessType, dbOpts)
	if serviceErr != nil {
		return serviceErr
	}
	if cfg.AutoMigrate {
		if migrationErr := migrations.Run(sqlDB, "migrations"); migrationErr != nil {
			return fmt.Errorf("run migrations: %w", migrationErr)
		}
	}

	httpHandler := httpserver.NewHandler(service)

	externalClient := external.NewHTTPClient(timeout, cfg.GitHubAPIBaseURL, cfg.StackExchangeAPIBaseURL)

	var updatesClient botclient.MessageSender
	switch cfg.NotificationMode {
	case config.NotificationModeKafka:
		writeTimeout, parseErr := time.ParseDuration(cfg.KafkaWriteTimeout)
		if parseErr != nil {
			return fmt.Errorf("parse kafka write timeout: %w", parseErr)
		}
		kafkaClient, kafkaErr := botclient.NewKafkaUpdatesClient(cfg.KafkaBrokers, cfg.KafkaTopic, writeTimeout, cfg.KafkaSchemaRegistryURL)
		if kafkaErr != nil {
			return fmt.Errorf("create kafka updates client: %w", kafkaErr)
		}
		updatesClient = kafkaClient
	default:
		updatesClient = botclient.NewHTTPUpdatesClient(cfg.BotBaseURL, timeout)
	}

	p := poller.New(service, externalClient, updatesClient, poller.Config{
		DBPageSize:     cfg.SchedulerDBPageSize,
		SuperBatchSize: cfg.SchedulerSuperBatchSize,
		WorkerCount:    cfg.SchedulerWorkerCount,
		CheckInterval:  interval,
	})

	a.config = cfg
	a.service = service
	a.httpHandler = httpHandler
	a.poller = p
	a.interval = interval
	a.sqlDB = sqlDB
	if cfg.NotificationMode == config.NotificationModeKafka {
		dispatchInterval, parseErr := time.ParseDuration(cfg.OutboxDispatchInterval)
		if parseErr != nil {
			return fmt.Errorf("parse outbox dispatch interval: %w", parseErr)
		}
		a.outbox = poller.NewOutboxDispatcher(service, updatesClient, 100, dispatchInterval)
		a.outboxTick = dispatchInterval
	}

	return nil
}

func (a *App) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		if a.sqlDB != nil {
			_ = a.sqlDB.Close()
		}
	}()

	go a.runScheduledChecks(ctx)

	go a.runGRPCServer(ctx)
	if a.outbox != nil {
		go a.runOutboxDispatcher(ctx)
	}

	slog.Info("Scrapper HTTP server is up", slog.String("address", a.config.ScrapperHTTPAddress))

	if err := http.ListenAndServe(a.config.ScrapperHTTPAddress, a.httpHandler.Handler()); err != nil {
		slog.Error("Scrapper HTTP server failed", slog.String("error", err.Error()))

		return
	}
}

func createRepositoryByAccessType(accessType string, opts db.Options) (repository.Service, *sql.DB, error) {
	switch {
	case strings.EqualFold(accessType, "orm"):
		gormDB, sqlDB, err := db.OpenGORM(opts)
		if err != nil {
			return nil, nil, fmt.Errorf("open orm db: %w", err)
		}
		return ormrepo.New(gormDB), sqlDB, nil
	case strings.EqualFold(accessType, "sql"):
		sqlDB, err := db.OpenSQL(opts)
		if err != nil {
			return nil, nil, fmt.Errorf("open sql db: %w", err)
		}
		return sqlrepo.New(sqlDB), sqlDB, nil
	default:
		return nil, nil, fmt.Errorf("unsupported access_type %q; expected SQL or ORM", accessType)
	}
}

func (a *App) runScheduledChecks(ctx context.Context) {
	cron := gocron.NewScheduler(time.UTC)
	_, err := cron.Every(a.interval).Do(func() {
		a.poller.ProcessOnce(ctx)
	})
	if err != nil {
		slog.Error("Failed to schedule link checks", slog.String("error", err.Error()))

		return
	}

	cron.StartAsync()

	<-ctx.Done()

	cron.Stop()
}

func (a *App) runOutboxDispatcher(ctx context.Context) {
	tick := a.outboxTick
	if tick <= 0 {
		tick = 2 * time.Second
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.outbox.DispatchOnce(ctx)
		}
	}
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
