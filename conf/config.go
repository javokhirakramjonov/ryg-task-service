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
	DB                DBConfig
	RYGTaskServiceUrl string
}

func LoadConfig() *Config {
	return &Config{
		DB: DBConfig{
			DBHost:     os.Getenv("POSTGRES_DB_HOST"),
			DBPort:     os.Getenv("POSTGRES_DB_PORT"),
			DBUser:     os.Getenv("POSTGRES_DB_USER"),
			DBPassword: os.Getenv("POSTGRES_DB_PASSWORD"),
			DBName:     os.Getenv("POSTGRES_DB_NAME"),
			SSLMode:    os.Getenv("POSTGRES_DB_SSL_MODE"),
			TimeZone:   os.Getenv("POSTGRES_DB_TIMEZONE"),
		},
		RYGTaskServiceUrl: os.Getenv("RYG_TASK_SERVICE_URL"),
	}
}
