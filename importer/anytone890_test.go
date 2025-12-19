package importer

import (
	"os"
	"testing"

	"codeplugs/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func setupAnyToneTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file:memdb_at890_imp?mode=memory&cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	err = db.AutoMigrate(&models.Channel{}, &models.Contact{}, &models.DigitalContact{}, &models.Zone{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	// Clear tables
	db.Exec("DELETE FROM channels")
	db.Exec("DELETE FROM contacts")
	db.Exec("DELETE FROM digital_contacts")
	db.Exec("DELETE FROM zones")
	return db
}

func TestImportAnyTone890Channels(t *testing.T) {
	db := setupAnyToneTestDB(t)

	// Sample from file content view in step 50
	// Header: "No.","Channel Name","Receive Frequency","Transmit Frequency","Channel Type",...
	// Content: "1","Channel 1","440.00000","440.00000","D-Digital",...
	content := `"No.","Channel Name","Receive Frequency","Transmit Frequency","Channel Type","Transmit Power","Band Width","RX Color Code","Slot","Contact/Talk Group","Scan List"
"1","ATCh1","440.00000","445.00000","D-Digital","High","12.5K","1","1","Contact A","ScanList 1"`

	tmpfile, err := os.CreateTemp("", "at890_ch_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	f, _ := os.Open(tmpfile.Name())
	defer f.Close()
	if err := ImportAnyTone890Channels(db, f); err != nil {
		t.Fatalf("ImportAnyTone890Channels failed: %v", err)
	}

	var ch models.Channel
	if err := db.First(&ch).Error; err != nil {
		t.Fatalf("Channel not found: %v", err)
	}
	if ch.Name != "ATCh1" {
		t.Errorf("Expected ATCh1, got %s", ch.Name)
	}
	if ch.Type != models.ChannelTypeDigitalDMR {
		t.Errorf("Expected Digital, got %s", ch.Type)
	}
	if ch.TimeSlot != 1 {
		t.Errorf("Expected TimeSlot 1, got %d", ch.TimeSlot)
	}
	if ch.TxContact != "Contact A" {
		t.Errorf("Expected Contact A, got %s", ch.TxContact)
	}
}

func TestImportAnyTone890Talkgroups(t *testing.T) {
	db := setupAnyToneTestDB(t)

	content := `"No.","Radio ID","Name","Call Type","Call Alert"
"1","101","TG Group","Group Call","None"
"2","102","Individual","Private Call","None"`

	tmpfile, err := os.CreateTemp("", "at890_tg_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	f, _ := os.Open(tmpfile.Name())
	defer f.Close()
	if err := ImportAnyTone890Talkgroups(db, f); err != nil {
		t.Fatalf("ImportAnyTone890Talkgroups failed: %v", err)
	}

	var count int64
	db.Model(&models.Contact{}).Count(&count)
	if count != 2 {
		t.Errorf("Expected 2 contacts, got %d", count)
	}
}

func TestImportAnyTone890Zones(t *testing.T) {
	db := setupAnyToneTestDB(t)
	db.Create(&models.Channel{Name: "ChA"})
	db.Create(&models.Channel{Name: "ChB"})

	// "No.","Zone Name","Zone Channel Member"
	// "1","ZoneX","ChA|ChB"
	content := `"No.","Zone Name","Zone Channel Member"
"1","ZoneX","ChA|ChB"`

	tmpfile, err := os.CreateTemp("", "at890_zone_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	f, _ := os.Open(tmpfile.Name())
	defer f.Close()
	if err := ImportAnyTone890Zones(db, f); err != nil {
		t.Fatalf("ImportAnyTone890Zones failed: %v", err)
	}

	var zone models.Zone
	if err := db.Preload("Channels").First(&zone).Error; err != nil {
		t.Fatal(err)
	}
	if zone.Name != "ZoneX" {
		t.Errorf("Expected ZoneX, got %s", zone.Name)
	}
	if len(zone.Channels) != 2 {
		t.Errorf("Expected 2 channels, got %d", len(zone.Channels))
	}
}
