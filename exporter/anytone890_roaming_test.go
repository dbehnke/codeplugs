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

func setupAnyToneRoamingExpDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file:memdb_at890_exp_roam?mode=memory&cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&models.RoamingChannel{}, &models.RoamingZone{})
	return db
}

func TestExportAnyTone890Roaming(t *testing.T) {
	db := setupAnyToneRoamingExpDB(t)
	// Create mock data
	rc := models.RoamingChannel{
		Name:        "RoamCh1",
		RxFrequency: 440.0,
		TxFrequency: 445.0,
		ColorCode:   1,
		TimeSlot:    1,
	}
	db.Create(&rc)

	tmpDir, err := os.MkdirTemp("", "at890_exp_roam_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Undefined function
	// We pass nil for models since they don't exist, this test will fail compilation on the function call
	if err := ExportAnyTone890Roaming(db, tmpDir); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify existence
	if _, err := os.Stat(filepath.Join(tmpDir, "RoamChannel.CSV")); os.IsNotExist(err) {
		t.Errorf("RoamChannel.CSV missing")
	}

	f, _ := os.Open(filepath.Join(tmpDir, "RoamChannel.CSV"))
	reader := csv.NewReader(f)
	records, _ := reader.ReadAll()
	if len(records) < 2 {
		t.Error("RoamChannel.CSV empty")
	}
}
