package models

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func connectDatabase() error {
	var err error

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("❌ Failed to connect to database: %v", err)
	}
	log.Println("✅ Connected to the database")
	return nil
}

func AutoMigrate() error {
	if err := connectDatabase(); err != nil {
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
