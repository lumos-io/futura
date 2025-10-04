package models

import (
	"fmt"
	"time"

	"github.com/Unleash/unleash-client-go/v4"
	"github.com/opisvigilant/futura/backend/internal/shared/config"
	"github.com/rs/zerolog/log"
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

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return fmt.Errorf("❌ Failed to connect to database: %v", err)
	}
	log.Logger.Info().Msg("✅ Connected to the database")
	return nil
}

func AutoMigrate(dbConfig *config.Database) error {
	if err := connectDatabase(dbConfig); err != nil {
		return err
	}

	if err := db.AutoMigrate(
		&Organization{},
		&User{},
		&Team{},
		&TeamMember{},
		&OAuthState{},
		&RefreshToken{},
		&AuditLog{},
		&ProviderConnection{},
		&ClusterMetadata{},
		&EKSClusterMetadata{},
		&GKEClusterMetadata{},
		&AKSClusterMetadata{},
		&DOKSClusterMetadata{},
		&ACKClusterMetadata{},
	); err != nil {
		return fmt.Errorf("❌ Failed to auto-migrate models: %v", err)
	}

	if unleash.IsEnabled("kind.cluster") {
		if err := db.AutoMigrate(&KindClusterMetadata{}); err != nil {
			return fmt.Errorf("❌ Failed to auto-migrate model: %v", err)
		}
	}

	log.Logger.Info().Msg("✅ Database migration complete")
	return nil
}

func GetDB() *gorm.DB {
	return db
}

// SetDB sets the database instance (for testing)
func SetDB(database *gorm.DB) {
	db = database
}

type BaseModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
