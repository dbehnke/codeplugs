package exporter

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"codeplugs/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:dm32uv_exp?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	err = db.AutoMigrate(&models.Channel{}, &models.Contact{}, &models.DigitalContact{}, &models.Zone{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	return db
}

func TestExportDM32UV(t *testing.T) {
	db := setupTestDB(t)

	// Seed data
	db.Create(&models.Channel{
		Name:        "TestCh1",
		RxFrequency: 440.0,
		TxFrequency: 445.0,
		Type:        models.ChannelTypeDigital,
		TxContact:   "TestGroup",
		RxGroup:     "TestGroup",
		TimeSlot:    1,
		ColorCode:   1,
	})

	db.Create(&models.Contact{
		Name:  "TestGroup",
		DMRID: 999,
		Type:  models.ContactTypeGroup,
	})

	zone := models.Zone{Name: "Zone1"}
	db.Create(&zone) // Create zone first

	var ch models.Channel
	db.First(&ch)
	db.Model(&zone).Association("Channels").Append(&ch)

	tmpDir, err := os.MkdirTemp("", "dm32uv_export_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	var count int64
	db.Model(&models.Channel{}).Count(&count)
	if count == 0 {
		t.Fatalf("Pre-export check: 0 channels in DB")
	}

	if err := ExportDM32UV(db, tmpDir); err != nil {
		t.Errorf("ExportDM32UV failed: %v", err)
	}

	// Verify channels.csv
	chanFile := filepath.Join(tmpDir, "channels.csv")
	if _, err := os.Stat(chanFile); os.IsNotExist(err) {
		t.Errorf("channels.csv not generated")
	} else {
		// Basic content check
		f, _ := os.Open(chanFile)
		reader := csv.NewReader(f)
		records, _ := reader.ReadAll()
		if len(records) < 2 {
			t.Errorf("channels.csv empty or header only")
		} else {
			if records[1][1] != "TestCh1" {
				t.Errorf("Expected TestCh1, got %s", records[1][1])
			}
		}
		f.Close()
	}

	// Verify talkgroups.csv
	tgFile := filepath.Join(tmpDir, "talkgroups.csv")
	if _, err := os.Stat(tgFile); os.IsNotExist(err) {
		t.Errorf("talkgroups.csv not generated")
	}

	// Verify zones.csv
	zoneFile := filepath.Join(tmpDir, "zones.csv")
	if _, err := os.Stat(zoneFile); os.IsNotExist(err) {
		t.Errorf("zones.csv not generated")
	}
}
