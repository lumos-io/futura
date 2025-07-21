package models

import (
	"fmt"
	"log"

	"github.com/opisvigilant/futura/apis/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func connectDatabase(dbConfig *config.Database) error {
	var err error

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.User,
		dbConfig.Password, dbConfig.Name, dbConfig.SSLMode,
	)

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("❌ Failed to connect to database: %v", err)
	}
	log.Println("✅ Connected to the database")
	return nil
}

func AutoMigrate(dbConfig *config.Database) error {
	if err := connectDatabase(dbConfig); err != nil {
		return err
	}

	err := db.AutoMigrate(
		&Organization{},
		&User{},
		&CloudProvider{},
		&ClusterMetadata{},
		&EKSClusterMetadata{},
		&GKEClusterMetadata{},
		&AKSClusterMetadata{},
		&DOKSClusterMetadata{},
		&ACKClusterMetadata{},
	)

	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_cluster_org_provider_name ON cluster_metadata (organization_id, cloud_provider_id);`)

	if err != nil {
		return fmt.Errorf("❌ Failed to auto-migrate models: %v", err)
	}

	log.Println("✅ Database migration complete")
	return nil
}

func GetDB() *gorm.DB {
	return db
}
