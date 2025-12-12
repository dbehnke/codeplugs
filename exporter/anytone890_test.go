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

func setupAnyToneTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:at890_exp?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	err = db.AutoMigrate(&models.Channel{}, &models.Contact{}, &models.DigitalContact{}, &models.Zone{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	return db
}

func TestExportAnyTone890(t *testing.T) {
	db := setupAnyToneTestDB(t)

	db.Create(&models.Channel{
		Name:        "TestCh890",
		RxFrequency: 145.500,
		TxFrequency: 145.500,
		Type:        models.ChannelTypeAnalog,
		Power:       "Low",
		Bandwidth:   "25K",
	})

	tmpDir, err := os.MkdirTemp("", "at890_export_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := ExportAnyTone890(db, tmpDir); err != nil {
		t.Fatalf("ExportAnyTone890 failed: %v", err)
	}

	// Verify Channel.CSV
	chanFile := filepath.Join(tmpDir, "Channel.CSV")
	if _, err := os.Stat(chanFile); os.IsNotExist(err) {
		t.Errorf("Channel.CSV not generated")
	} else {
		f, _ := os.Open(chanFile)
		reader := csv.NewReader(f)
		records, _ := reader.ReadAll()
		if len(records) < 2 {
			t.Errorf("Channel.CSV empty")
		} else {
			// Check header count
			if len(records[0]) < 70 {
				t.Errorf("Channel.CSV header seems too short: %d", len(records[0]))
			}
			if records[1][1] != "TestCh890" {
				t.Errorf("Expected TestCh890, got %s", records[1][1])
			}
			if records[1][4] != "A-Analog" {
				t.Errorf("Expected A-Analog, got %s", records[1][4])
			}
		}
		f.Close()
	}

	// Check other files existence
	if _, err := os.Stat(filepath.Join(tmpDir, "DMRTalkGroups.CSV")); os.IsNotExist(err) {
		t.Errorf("DMRTalkGroups.CSV not generated")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "DMRZone.CSV")); os.IsNotExist(err) {
		t.Errorf("DMRZone.CSV not generated")
	}
}
