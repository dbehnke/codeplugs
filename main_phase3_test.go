package main

import (
	"bytes"
	"codeplugs/api"
	"codeplugs/database"
	"codeplugs/exporter"
	"codeplugs/importer"
	"codeplugs/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPhase3DB(t *testing.T) {
	var err error
	database.DB, err = gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file:memdb_phase3?mode=memory&cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	database.DB.SetupJoinTable(&models.Zone{}, "Channels", &models.ZoneChannel{})
	err = database.DB.AutoMigrate(&models.Channel{}, &models.Zone{}, &models.Contact{}, &models.ZoneChannel{}, &models.DigitalContact{}, &models.ScanList{}, &models.RoamingChannel{}, &models.RoamingZone{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	database.DB.Exec("DELETE FROM zone_channels")
	database.DB.Exec("DELETE FROM channels")
}

func TestPhase3_APIJSONError(t *testing.T) {
	setupPhase3DB(t)

	// Send invalid JSON to Channel Reorder to trigger error
	req, _ := http.NewRequest("POST", "/api/channels/reorder", bytes.NewBufferString("{invalid_json"))
	rr := httptest.NewRecorder()
	http.HandlerFunc(api.HandleChannelReorder).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", rr.Code)
	}

	// Verify Content-Type is JSON
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", ct)
	}

	// Verify body is JSON object with error field
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Response is not valid JSON: %v. Body: %s", err, rr.Body.String())
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Response JSON missing 'error' field")
	}
}

func TestPhase3_APIStandardization(t *testing.T) {
	setupPhase3DB(t)

	// Create a channel to ensure data exists
	ch := models.Channel{Name: "API Test Channel", RxFrequency: 146.52}
	database.DB.Create(&ch)

	req, _ := http.NewRequest("GET", "/api/channels", nil)
	rr := httptest.NewRecorder()
	http.HandlerFunc(api.HandleChannels).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}

	// Verify response structure
	var resp struct {
		Success bool             `json:"success"`
		Data    []models.Channel `json:"data"`
		Error   string           `json:"error"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Response is not valid JSON: %v. Body: %s", err, rr.Body.String())
	}

	if !resp.Success {
		t.Error("Expected Success to be true")
	}

	if len(resp.Data) != 1 {
		t.Errorf("Expected 1 channel in Data, got %d", len(resp.Data))
	}
	
	if resp.Data[0].Name != "API Test Channel" {
		t.Errorf("Expected channel name 'API Test Channel', got '%s'", resp.Data[0].Name)
	}
}

func TestPhase3_ScanListLogic(t *testing.T) {
	setupPhase3DB(t)
	database.DB.AutoMigrate(&models.ScanList{})

	sl := models.ScanList{Name: "TestList"}
	if err := database.DB.Create(&sl).Error; err != nil {
		t.Fatalf("Failed to create ScanList: %v", err)
	}

	ch := models.Channel{Name: "Ch1"}
	database.DB.Create(&ch)

	// Link
	database.DB.Model(&sl).Association("Channels").Append(&ch)

	count := database.DB.Model(&sl).Association("Channels").Count()
	if count != 1 {
		t.Errorf("Expected 1 channel in scan list, got %d", count)
	}
}

// TODO: Add Roaming tests once models are created

func TestPhase3_Integration_RoundTrip(t *testing.T) {
	setupPhase3DB(t)

	// 1. Create Data
	ch1 := models.Channel{Name: "Rpt 1", RxFrequency: 440.000}
	ch2 := models.Channel{Name: "Rpt 2", RxFrequency: 145.000}
	database.DB.Create(&ch1)
	database.DB.Create(&ch2)

	sl := models.ScanList{Name: "Scan A"}
	database.DB.Create(&sl)
	database.DB.Model(&sl).Association("Channels").Append(&ch1)
	database.DB.Model(&sl).Association("Channels").Append(&ch2)

	rz := models.RoamingZone{Name: "Roam Z"}
	database.DB.Create(&rz) // Just empty or with channels
	rc1 := models.RoamingChannel{Name: "Site 1", RxFrequency: 442.000, TxFrequency: 447.000, ColorCode: 1, TimeSlot: 2}
	database.DB.Create(&rc1)
	database.DB.Model(&rz).Association("Channels").Append(&rc1)

	// 2. Export AnyTone 890
	tempDir890 := t.TempDir()
	if err := exporter.ExportAnyTone890(database.DB, tempDir890, 0); err != nil {
		t.Fatalf("ExportAnyTone890 failed: %v", err)
	}

	// 3. Clear DB (simulating fresh import)
	database.DB.Exec("DELETE FROM scan_list_channels")
	database.DB.Exec("DELETE FROM scan_lists")
	database.DB.Exec("DELETE FROM roaming_zone_channels")
	database.DB.Exec("DELETE FROM roaming_zones")
	database.DB.Exec("DELETE FROM roaming_channels")
	// Channels kept? Or also cleared? Import usually expects Channels to exist for ScanLists referencing them.
	// But ImportAnyTone890ScanList looks for channels by name.
	// The Export generated Channel.CSV too.
	// So we should verify we can re-import everything or assumes channels exist.
	// Let's assume channels exist for now or re-import Channels too if we implemented that (we did).

	// 4. Import AnyTone 890 Scan Lists
	f, err := os.Open(filepath.Join(tempDir890, "ScanList.CSV"))
	if err != nil {
		t.Fatalf("Failed to open exported ScanList.CSV: %v", err)
	}
	defer f.Close()
	if err := importer.ImportAnyTone890ScanLists(database.DB, f); err != nil {
		t.Fatalf("ImportAnyTone890ScanLists failed: %v", err)
	}

	// Verify Scan List
	var importedSL models.ScanList
	if err := database.DB.Preload("Channels").First(&importedSL, "name = ?", "Scan A").Error; err != nil {
		t.Fatalf("Failed to find imported Scan List 'Scan A': %v", err)
	}
	if len(importedSL.Channels) != 2 {
		t.Errorf("Expected 2 channels in imported scan list, got %d", len(importedSL.Channels))
	}

	// 5. Import AnyTone 890 Roaming
	f2, err := os.Open(filepath.Join(tempDir890, "RoamChannel.CSV"))
	if err == nil {
		defer f2.Close()
		if err := importer.ImportAnyTone890RoamingChannels(database.DB, f2); err != nil {
			t.Fatalf("ImportAnyTone890RoamingChannels failed: %v", err)
		}
	} else {
		t.Fatal("RoamChannel.CSV not found")
	}

	f3, err := os.Open(filepath.Join(tempDir890, "RoamZone.CSV"))
	if err == nil {
		defer f3.Close()
		if err := importer.ImportAnyTone890RoamingZones(database.DB, f3); err != nil {
			t.Fatalf("ImportAnyTone890RoamingZones failed: %v", err)
		}
	} else {
		t.Fatal("RoamZone.CSV not found")
	}

	// Verify Roaming
	var importedRZ models.RoamingZone
	if err := database.DB.Preload("Channels").First(&importedRZ, "name = ?", "Roam Z").Error; err != nil {
		t.Fatalf("Failed to find imported Roaming Zone 'Roam Z': %v", err)
	}
	if len(importedRZ.Channels) != 1 {
		t.Errorf("Expected 1 channel in roaming zone, got %d", len(importedRZ.Channels))
	}
	if importedRZ.Channels[0].Name != "Site 1" {
		t.Errorf("Expected channel 'Site 1', got %s", importedRZ.Channels[0].Name)
	}
}

func TestPhase3_DM32UV_RoundTrip(t *testing.T) {
	setupPhase3DB(t)

	// 1. Create Data
	ch1 := models.Channel{Name: "Rpt A", RxFrequency: 443.000}
	ch2 := models.Channel{Name: "Rpt B", RxFrequency: 147.000}
	database.DB.Create(&ch1)
	database.DB.Create(&ch2)

	sl := models.ScanList{Name: "Scan List 1"}
	database.DB.Create(&sl)
	database.DB.Model(&sl).Association("Channels").Append(&ch1)
	database.DB.Model(&sl).Association("Channels").Append(&ch2)

	rz := models.RoamingZone{Name: "Roam Zone 1"}
	database.DB.Create(&rz) 
	rc1 := models.RoamingChannel{Name: "Roam Ch 1", RxFrequency: 444.000, TxFrequency: 449.000, ColorCode: 1, TimeSlot: 1}
	database.DB.Create(&rc1)
	database.DB.Model(&rz).Association("Channels").Append(&rc1)

	// 2. Export DM32UV
	tempDirDM32UV := t.TempDir()
	if err := exporter.ExportDM32UV(database.DB, tempDirDM32UV); err != nil {
		t.Fatalf("ExportDM32UV failed: %v", err)
	}

	// 3. Clear DB
	database.DB.Exec("DELETE FROM scan_list_channels")
	database.DB.Exec("DELETE FROM scan_lists")
	database.DB.Exec("DELETE FROM roaming_zone_channels")
	database.DB.Exec("DELETE FROM roaming_zones")
	database.DB.Exec("DELETE FROM roaming_channels")

	// 4. Import DM32UV Scan Lists
	f, err := os.Open(filepath.Join(tempDirDM32UV, "scan_lists.csv"))
	if err != nil {
		t.Fatalf("Failed to open exported scan_lists.csv: %v", err)
	}
	defer f.Close()
	if err := importer.ImportDM32UVScanLists(database.DB, f); err != nil {
		t.Fatalf("ImportDM32UVScanLists failed: %v", err)
	}

	// Verify Scan List
	var importedSL models.ScanList
	if err := database.DB.Preload("Channels").First(&importedSL, "name = ?", "Scan List 1").Error; err != nil {
		t.Fatalf("Failed to find imported Scan List 'Scan List 1': %v", err)
	}
	if len(importedSL.Channels) != 2 {
		t.Errorf("Expected 2 channels in imported scan list, got %d", len(importedSL.Channels))
	}

	// 5. Import DM32UV Roaming
	f2, err := os.Open(filepath.Join(tempDirDM32UV, "roaming_channels.csv"))
	if err == nil {
		defer f2.Close()
		if err := importer.ImportDM32UVRoamingChannels(database.DB, f2); err != nil {
			t.Fatalf("ImportDM32UVRoamingChannels failed: %v", err)
		}
	} else {
		t.Fatal("roaming_channels.csv not found")
	}

	f3, err := os.Open(filepath.Join(tempDirDM32UV, "roaming_zones.csv"))
	if err == nil {
		defer f3.Close()
		if err := importer.ImportDM32UVRoamingZones(database.DB, f3); err != nil {
			t.Fatalf("ImportDM32UVRoamingZones failed: %v", err)
		}
	} else {
		t.Fatal("roaming_zones.csv not found")
	}

	// Verify Roaming
	var importedRZ models.RoamingZone
	if err := database.DB.Preload("Channels").First(&importedRZ, "name = ?", "Roam Zone 1").Error; err != nil {
		t.Fatalf("Failed to find imported Roaming Zone 'Roam Zone 1': %v", err)
	}
	if len(importedRZ.Channels) != 1 {
		t.Errorf("Expected 1 channel in roaming zone, got %d", len(importedRZ.Channels))
	}
	if importedRZ.Channels[0].Name != "Roam Ch 1" {
		t.Errorf("Expected channel 'Roam Ch 1', got %s", importedRZ.Channels[0].Name)
	}
}
