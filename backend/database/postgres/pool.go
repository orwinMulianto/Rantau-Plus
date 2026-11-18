package postgres

import (
	"context"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(ctx context.Context, connectionURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(connectionURL), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, err
	}
	return db, nil
}
