package api

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("./gorm.db"), &gorm.Config{})
	if err != nil {
		log.Printf("db err: (Init) %v", err)
	}
	if sqlDb, err := db.DB(); err == nil {
		sqlDb.SetMaxIdleConns(10)
	}

	//db.LogMode(true)
	return db
}

func CloseDB(db *gorm.DB) {
	// Close the database connection when main exits
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting underlying SQL DB: %v", err)
	}
	if err = sqlDB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
}
