package conf

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type DBConfig struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	SSLMode    string
	TimeZone   string
}

type Config struct {
	DB      DBConfig
	GRPCUrl string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	return &Config{
		DB: DBConfig{
			DBHost:     os.Getenv("DB_HOST"),
			DBPort:     os.Getenv("DB_PORT"),
			DBUser:     os.Getenv("DB_USER"),
			DBPassword: os.Getenv("DB_PASSWORD"),
			DBName:     os.Getenv("DB_NAME"),
			SSLMode:    os.Getenv("DB_SSL_MODE"),
			TimeZone:   os.Getenv("DB_TIMEZONE"),
		},
		GRPCUrl: os.Getenv("GRPC_URL"),
	}
}
