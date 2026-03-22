package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" 
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Options struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func OpenSQL(opts Options) (*sql.DB, error) {
	db, err := sql.Open("pgx", opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("open sql db: %w", err)
	}
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(opts.ConnMaxLifetime)
	if pingErr := db.PingContext(context.Background()); pingErr != nil {
		return nil, fmt.Errorf("ping sql db: %w", pingErr)
	}
	return db, nil
}

func OpenGORM(opts Options) (*gorm.DB, *sql.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(opts.DSN), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("open gorm db: %w", err)
	}
	sqlDB, dbErr := gormDB.DB()
	if dbErr != nil {
		return nil, nil, fmt.Errorf("gorm underlying sql db: %w", dbErr)
	}
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)
	if pingErr := sqlDB.PingContext(context.Background()); pingErr != nil {
		return nil, nil, fmt.Errorf("ping gorm sql db: %w", pingErr)
	}
	return gormDB, sqlDB, nil
}
