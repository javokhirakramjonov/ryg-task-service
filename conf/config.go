package conf

import (
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
