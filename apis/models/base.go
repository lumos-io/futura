package models

import (
	"fmt"
	"log"

	"github.com/opisvigilant/futura/apis/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func connectDatabase(apisCfg *config.Configuration) error {
	var err error

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		apisCfg.Database.Host, apisCfg.Database.Port, apisCfg.Database.User,
		apisCfg.Database.Password, apisCfg.Database.Name, apisCfg.Database.SSLMode,
	)

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("❌ Failed to connect to database: %v", err)
	}
	log.Println("✅ Connected to the database")
	return nil
}

func AutoMigrate(apisCfg *config.Configuration) error {
	if err := connectDatabase(apisCfg); err != nil {
		return err
	}

	err := db.AutoMigrate(
		&Organization{},
		&User{},
		&CloudProvider{},
		&Cluster{},
	)

	if err != nil {
		return fmt.Errorf("❌ Failed to auto-migrate models: %v", err)
	}

	log.Println("✅ Database migration complete")
	return nil
}

func GetDB() *gorm.DB {
	return db
}
