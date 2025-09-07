package db

import (
	"os"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init initializes the database connection using environment variables.
// It reads the connection string from the DATABASE_URL environment variable
// and returns the connected *gorm.DB instance. The connection is also stored
// in the package-level DB variable for global access.
func Init() *gorm.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Warn().Msg("DATABASE_URL not set")
		return nil
	}

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect database")
	}

	DB = database
	return DB
}
