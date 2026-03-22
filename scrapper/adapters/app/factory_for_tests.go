//go:build integration

package app

import (
	"database/sql"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/db"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func CreateRepositoryForTests(accessType string, opts db.Options) (repository.Service, *sql.DB, error) {
	return createRepositoryByAccessType(accessType, opts)
}
