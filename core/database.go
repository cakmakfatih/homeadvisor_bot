package core

import (
	"homeadvisorbot/models"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	Db *gorm.DB
}

func NewDatabase() *Database {
	db, err := gorm.Open(sqlite.Open("homeadvisor.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		log.Fatal("could not connect to the db")
	}

	db.AutoMigrate(&models.HomeadvisorListingModel{})

	return &Database{
		Db: db,
	}
}
