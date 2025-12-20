package exporter

import (
	"codeplugs/models"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAnyToneScanListDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file:memdb_at890_exp_sl?mode=memory&cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&models.Channel{}, &models.ScanList{})
	return db
}

func TestExportAnyTone890ScanLists(t *testing.T) {
	db := setupAnyToneScanListDB(t)

	// Create Data
	c1 := models.Channel{Name: "Ch1"}
	c2 := models.Channel{Name: "Ch2"}
	db.Create(&c1)
	db.Create(&c2)

	sl := models.ScanList{Name: "List A"}
	db.Create(&sl)
	db.Model(&sl).Association("Channels").Append(&c1)
	db.Model(&sl).Association("Channels").Append(&c2)

	tmpDir, err := os.MkdirTemp("", "at890_exp_sl_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	f, err := os.Create(filepath.Join(tmpDir, "ScanList.CSV"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	scanLists := []models.ScanList{sl}
	// Need to reload to get channels
	db.Preload("Channels").Find(&scanLists)

	if err := ExportAnyTone890ScanLists(scanLists, f); err != nil {
		t.Fatalf("ExportAnyTone890ScanLists failed: %v", err)
	}

	// Verify
	f.Close()
	f, _ = os.Open(filepath.Join(tmpDir, "ScanList.CSV"))
	reader := csv.NewReader(f)
	records, _ := reader.ReadAll()

	if len(records) != 2 { // Header + 1 record
		t.Errorf("Expected 2 records, got %d", len(records))
	}
	if records[1][1] != "List A" {
		t.Errorf("Expected List A, got %s", records[1][1])
	}
	// Check members (index 2)
	// Expect "Ch1|Ch2" or similar
}
