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

type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

type Config struct {
	DB                DBConfig
	RabbitMQConfig    RabbitMQConfig
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
		RabbitMQConfig: RabbitMQConfig{
			Host:     os.Getenv("RABBITMQ_HOST"),
			Port:     os.Getenv("RABBITMQ_PORT"),
			User:     os.Getenv("RABBITMQ_USER"),
			Password: os.Getenv("RABBITMQ_PASSWORD"),
		},
		RYGTaskServiceUrl: os.Getenv("RYG_TASK_SERVICE_URL"),
	}
}
