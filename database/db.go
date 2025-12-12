package database

import (
	"log"

	"codeplugs/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // Register modernc sqlite driver
)

var DB *gorm.DB

func Connect(dbPath string) {
	var err error
	DB, err = gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)",
	}, &gorm.Config{})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Register Join Table for Ordering
	DB.SetupJoinTable(&models.Zone{}, "Channels", &models.ZoneChannel{})

	// Auto Migrate
	err = DB.AutoMigrate(&models.Channel{}, &models.Contact{}, &models.Zone{}, &models.DigitalContact{}, &models.ZoneChannel{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}
