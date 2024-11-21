package db

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"ryg-task-service/conf"
	"ryg-task-service/model"
)

var DB *gorm.DB

func ConnectDB(cnf conf.DBConfig) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cnf.DBHost,
		cnf.DBUser,
		cnf.DBPassword,
		cnf.DBName,
		cnf.DBPort,
		cnf.SSLMode,
		cnf.TimeZone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	DB = db
	fmt.Println("Connected to the database")

	if err := DB.AutoMigrate(&model.Challenge{}, &model.Task{}, &model.TaskAndStatus{}); err != nil {
		log.Fatalf("Error migrating database: %v", err)
	}
	fmt.Println("Database migrated")
}

func CloseDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Error getting SQL db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		log.Fatalf("Error closing the database: %v", err)
	}
	fmt.Println("Database connection closed")
}
