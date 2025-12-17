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
		DSN:        dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
	}, &gorm.Config{
		// Disable built-in foreign key constraints during AutoMigrate if we want to rely on SQLite's ON UPDATE CASCADE
		// But usually GORM handles migrations well.
		// For Sort/Reorder we need CASCADE updates.
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Register Join Table for Ordering
	DB.SetupJoinTable(&models.Zone{}, "Channels", &models.ZoneChannel{})
	DB.SetupJoinTable(&models.ScanList{}, "Channels", &models.ScanListChannel{})

	// Auto Migrate
	err = DB.AutoMigrate(&models.Channel{}, &models.Contact{}, &models.Zone{}, &models.DigitalContact{}, &models.ZoneChannel{}, &models.ScanList{}, &models.ScanListChannel{}, &models.ContactList{}, &models.ContactListEntry{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}

func Close() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			log.Println("Error getting SQL DB from GORM:", err)
			return
		}
		sqlDB.Close()
	}
}
