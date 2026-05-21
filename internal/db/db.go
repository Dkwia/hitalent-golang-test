package db

import (
	"fmt"
	"log"

	"organization-api/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return gdb, nil
}

func AutoMigrate(gdb *gorm.DB) error {
	if err := gdb.AutoMigrate(&models.Department{}, &models.Employee{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	log.Println("database models migrated")
	return nil
}
