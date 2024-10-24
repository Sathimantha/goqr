package secondaryfunctions

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// DBConfig holds the database configuration details
var DBConfig struct {
	Username string
	Password string
	Host     string
	Port     string
	Database string
}

func init() {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Assign environment variables to DBConfig
	DBConfig = struct {
		Username string
		Password string
		Host     string
		Port     string
		Database string
	}{
		Username: os.Getenv("DB_USERNAME"),
		Password: os.Getenv("DB_PASSWORD"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Database: os.Getenv("DB_NAME"),
	}
}
