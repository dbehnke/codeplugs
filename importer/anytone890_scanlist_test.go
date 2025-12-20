package importer

import (
	"codeplugs/models"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAnyToneScanListDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file:memdb_at890_sl?mode=memory&cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&models.Channel{}, &models.ScanList{})
	return db
}

func TestImportAnyTone890ScanLists(t *testing.T) {
	db := setupAnyToneScanListDB(t)

	// Mock Channels
	db.Create(&models.Channel{Name: "Ch1"})
	db.Create(&models.Channel{Name: "Ch2"})

	content := `"No.","Scan List Name","Scan Channel Member"
"1","List A","Ch1|Ch2"
"2","List B","Ch1"`

	tmpfile, err := os.CreateTemp("", "at890_sl_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	f, _ := os.Open(tmpfile.Name())
	defer f.Close()

	if err := ImportAnyTone890ScanLists(db, f); err != nil {
		t.Fatalf("ImportAnyTone890ScanLists failed: %v", err)
	}

	var sl models.ScanList
	if err := db.Preload("Channels").Where("name = ?", "List A").First(&sl).Error; err != nil {
		t.Fatalf("List A not found: %v", err)
	}
	if len(sl.Channels) != 2 {
		t.Errorf("Expected 2 channels in List A, got %d", len(sl.Channels))
	}
}
