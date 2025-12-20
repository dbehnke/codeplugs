package importer

import (
	"codeplugs/models"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDM32UVRoamingDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file:memdb_dm32uv_roam?mode=memory&cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&models.RoamingChannel{}, &models.RoamingZone{})
	return db
}

func TestImportDM32UVRoaming(t *testing.T) {
	db := setupDM32UVRoamingDB(t)

	// Content based on dm32uv/roam-channels.csv
	// No.,Channel Name,RX Frequency,TX Frequency,Color Code,Time Slot
	// 1,Bancroft,443.31250,448.31250,1,1
	contentChan := `No.,Channel Name,RX Frequency,TX Frequency,Color Code,Time Slot
1,Bancroft,443.31250,448.31250,1,1`

	tmpfile, _ := os.CreateTemp("", "dm32uv_roam_ch_*.csv")
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(contentChan))
	tmpfile.Close()

	f, _ := os.Open(tmpfile.Name())
	defer f.Close()

	// Undefined function
	if err := ImportDM32UVRoamingChannels(db, f); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify manually via SQL
	var name string
	row := db.Raw("SELECT name FROM roaming_channels WHERE name = ?", "Bancroft").Row()
	row.Scan(&name)
	if name != "Bancroft" {
		t.Errorf("Expected Bancroft, got %s", name)
	}
}

func TestImportDM32UVRoamingZones(t *testing.T) {
	db := setupDM32UVRoamingDB(t)

	// Content based on dm32uv/roam-zones.csv
	// No.,Zone Name,Channel Members
	// 1,Flint/Saginaw,Bancroft|Bay City
	content := `No.,Zone Name,Channel Members
1,Flint/Saginaw,Bancroft|Bay City`

	tmpfile, _ := os.CreateTemp("", "dm32uv_roam_zone_*.csv")
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	f, _ := os.Open(tmpfile.Name())
	defer f.Close()

	// Undefined function
	if err := ImportDM32UVRoamingZones(db, f); err != nil {
		t.Fatalf("Import failed: %v", err)
	}
}
