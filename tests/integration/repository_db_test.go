//go:build integration

package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/app"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/db"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/db/migrations"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository/ormrepo"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository/sqlrepo"
)

func TestMigrationsApplyOnCleanDatabase(t *testing.T) {
	t.Parallel()
	requireTestcontainers(t)

	sqlDB, _, cleanup := startPostgresDB(t)
	defer cleanup()

	require.NoError(t, migrations.Run(sqlDB, "../../migrations"))

	tables := []string{"chats", "links", "chat_links", "tags", "chat_link_tags"}
	for _, table := range tables {
		var exists bool
		err := sqlDB.QueryRow(`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)`, table).Scan(&exists)
		require.NoError(t, err)
		require.True(t, exists, "table should exist after migrations: %s", table)
	}
}

func TestRepositoryContracts_SQLAndORM(t *testing.T) {
	t.Parallel()
	requireTestcontainers(t)

	t.Run("sql", func(t *testing.T) {
		t.Parallel()
		sqlDB, _, cleanup := startPostgresDB(t)
		defer cleanup()
		require.NoError(t, migrations.Run(sqlDB, "../../migrations"))
		runRepositoryContractScenarios(t, sqlrepo.New(sqlDB))
	})

	t.Run("orm", func(t *testing.T) {
		t.Parallel()
		sqlDB, dsn, cleanup := startPostgresDB(t)
		defer cleanup()
		require.NoError(t, migrations.Run(sqlDB, "../../migrations"))

		gormDB, _, err := db.OpenGORM(db.Options{
			DSN:             dsn,
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
		})
		require.NoError(t, err)

		runRepositoryContractScenarios(t, ormrepo.New(gormDB))
	})
}

func TestAccessTypeSwitchUsesExpectedImplementation(t *testing.T) {
	t.Parallel()
	requireTestcontainers(t)

	_, dsn, cleanup := startPostgresDB(t)
	defer cleanup()

	opts := db.Options{
		DSN:             dsn,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	}

	svcSQL, closerSQL, err := app.CreateRepositoryForTests("SQL", opts)
	require.NoError(t, err)
	require.NotNil(t, svcSQL)
	require.NotNil(t, closerSQL)
	require.IsType(t, &sqlrepo.Repository{}, svcSQL)
	require.NoError(t, closerSQL.Close())

	svcORM, closerORM, ormErr := app.CreateRepositoryForTests("ORM", opts)
	require.NoError(t, ormErr)
	require.NotNil(t, svcORM)
	require.NotNil(t, closerORM)
	require.IsType(t, &ormrepo.Repository{}, svcORM)
	require.NoError(t, closerORM.Close())
}

func runRepositoryContractScenarios(t *testing.T, svc repository.Service) {
	t.Helper()

	const chatID int64 = 101
	const rawURL = "https://github.com/example/repo"

	require.NoError(t, svc.RegisterChat(chatID))

	added, addErr := svc.AddLink(chatID, trackerapi.AddLinkRequest{
		Link:    rawURL,
		Tags:    []string{"work", "backend"},
		Filters: []string{"lang=go"},
	})
	require.NoError(t, addErr)
	require.Equal(t, rawURL, added.URL)
	require.NotEmpty(t, added.ID)

	_, duplicateErr := svc.AddLink(chatID, trackerapi.AddLinkRequest{Link: rawURL})
	require.Error(t, duplicateErr)
	require.True(t, errors.Is(duplicateErr, repository.ErrLinkExists))

	listed, listErr := svc.ListLinks(chatID)
	require.NoError(t, listErr)
	require.Equal(t, int32(1), listed.Size)
	require.Len(t, listed.Links, 1)
	require.Equal(t, rawURL, listed.Links[0].URL)

	removed, removeErr := svc.RemoveLink(chatID, trackerapi.RemoveLinkRequest{Link: rawURL})
	require.NoError(t, removeErr)
	require.Equal(t, rawURL, removed.URL)

	afterDelete, afterDeleteErr := svc.ListLinks(chatID)
	require.NoError(t, afterDeleteErr)
	require.Equal(t, int32(0), afterDelete.Size)

	createdTag, createTagErr := svc.CreateTag(chatID, "critical")
	require.NoError(t, createTagErr)
	require.Equal(t, "critical", createdTag.Name)

	tags, tagsErr := svc.ListTags(chatID, 10, 0)
	require.NoError(t, tagsErr)
	require.NotEmpty(t, tags)

	updatedTag, updateErr := svc.UpdateTag(chatID, "critical", "urgent")
	require.NoError(t, updateErr)
	require.Equal(t, "urgent", updatedTag.Name)

	require.NoError(t, svc.DeleteTag(chatID, "urgent"))
}

func startPostgresDB(t *testing.T) (*sql.DB, string, func()) {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "link_tracker",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "docker_engine") || strings.Contains(strings.ToLower(err.Error()), "must be run with elevated privileges") {
			t.Skipf("docker is unavailable for integration test: %v", err)
		}
		require.NoError(t, err)
	}

	host, hostErr := container.Host(ctx)
	require.NoError(t, hostErr)
	port, portErr := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, portErr)

	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/link_tracker?sslmode=disable", host, port.Port())
	sqlDB, dbErr := db.OpenSQL(db.Options{
		DSN:             dsn,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	})
	require.NoError(t, dbErr)

	return sqlDB, dsn, func() {
		_ = sqlDB.Close()
		_ = container.Terminate(ctx)
	}
}
