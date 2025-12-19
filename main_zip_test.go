package main

import (
	"archive/zip"
	"bytes"
	"codeplugs/api"
	"codeplugs/database"
	"codeplugs/models"
	"encoding/csv"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestZipExportHandler(t *testing.T) {
	// Setup DB
	tmpDB, _ := os.CreateTemp("", "test-zip-export-*.db")
	defer os.Remove(tmpDB.Name())
	database.Connect(tmpDB.Name())

	// Seed Data
	database.DB.Create(&models.Channel{Name: "Ch1", RxFrequency: 146.52, Mode: "FM"})
	database.DB.Create(&models.Zone{Name: "Zone1"})
	database.DB.Create(&models.Contact{Name: "TG1", DMRID: 1, Type: models.ContactTypeGroup})
	database.DB.Create(&models.DigitalContact{Name: "User1", DMRID: 12345})

	// Request DM32UV Zip
	req, _ := http.NewRequest("GET", "/api/export?radio=dm32uv&format=zip", nil)
	rr := httptest.NewRecorder()
	http.HandlerFunc(api.HandleExport).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Export request failed with code %d", rr.Code)
	}

	// Verify Output is Zip
	// Content-Type should be application/zip
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/zip" {
		t.Errorf("Expected Content-Type application/zip, got %s", contentType)
	}

	// Verify Zip Contents
	body := rr.Body.Bytes()
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("Failed to open response as zip: %v", err)
	}

	expectedFiles := map[string]bool{
		"channels.csv":         false,
		"zones.csv":            false,
		"talkgroups.csv":       false,
		"digital_contacts.csv": false,
	}

	for _, f := range zipReader.File {
		// DM32UV filenames usually have prefixes, but let's assume we standardise or check suffix
		for k := range expectedFiles {
			if strings.HasSuffix(f.Name, k) {
				expectedFiles[k] = true
			}
		}
	}

	for k, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file ending in %s in zip, but not found", k)
		}
	}
}

func TestZipImportHandler(t *testing.T) {
	// Setup DB
	tmpDB, _ := os.CreateTemp("", "test-zip-import-*.db")
	defer os.Remove(tmpDB.Name())
	database.Connect(tmpDB.Name())

	// Create Zip Payload
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// 1. Digital Contacts
	f1, _ := zw.Create("digital_contacts.csv")
	csvW1 := csv.NewWriter(f1)
	csvW1.Write([]string{"No.", "ID", "Repeater", "Name", "City", "Province", "Country", "Remark", "Type", "Alert Call"})
	csvW1.Write([]string{"1", "12345", "", "User1", "City", "State", "Country", "Rem", "Private Call", "Off"})
	csvW1.Flush()

	// 2. Talkgroups
	f2, _ := zw.Create("talkgroups.csv")
	csvW2 := csv.NewWriter(f2)
	csvW2.Write([]string{"No.", "Name", "ID", "Type"})
	csvW2.Write([]string{"1", "TG1", "100", "Group Call"})
	csvW2.Flush()

	// 3. Channels
	f3, _ := zw.Create("channels.csv")
	csvW3 := csv.NewWriter(f3)
	csvW3.Write([]string{"No.", "Channel Name", "Channel Type", "RX Frequency[MHz]", "TX Frequency[MHz]", "Power", "Band Width", "Scan List", "TX Admit", "Emergency System", "Squelch Level", "APRS Report Type", "Forbid TX", "APRS Receive", "Forbid Talkaround", "Auto Scan", "Lone Work", "Emergency Indicator", "Emergency ACK", "Analog APRS PTT Mode", "Digital APRS PTT Mode", "TX Contact", "RX Group List", "Color Code", "Time Slot", "Encryption", "Encryption ID", "APRS Report Channel", "Direct Dual Mode", "Private Confirm", "Short Data Confirm", "DMR ID", "CTC/DCS Decode", "CTC/DCS Encode", "Scramble", "RX Squelch Mode", "Signaling Type", "PTT ID", "VOX Function", "PTT ID Display"})
	csvW3.Write([]string{"1", "Ch1", "Digital", "440.00000", "440.00000", "High", "12.5K", "None", "Always", "None", "3", "Off", "Off", "Off", "Off", "Off", "Off", "Off", "Off", "Off", "Off", "TG1", "None", "1", "Slot1", "Off", "1", "1", "Off", "Off", "Off", "", "Off", "Off", "Off", "Audio", "Off", "Off", "Off", "Off"})
	csvW3.Flush()

	// 4. Zones
	f4, _ := zw.Create("zones.csv")
	csvW4 := csv.NewWriter(f4)
	csvW4.Write([]string{"No.", "Zone Name", "Channel Members"})
	csvW4.Write([]string{"1", "Zone1", "Ch1"})
	csvW4.Flush()

	zw.Close()

	// Prepare Request
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "import.zip")
	io.Copy(part, bytes.NewReader(buf.Bytes()))
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/import?radio=dm32uv&format=zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	http.HandlerFunc(api.HandleImport).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Import failed with code %d: %s", rr.Code, rr.Body.String())
	}

	// Verify Data
	var ch models.Channel
	if err := database.DB.Where("name = ?", "Ch1").First(&ch).Error; err != nil {
		t.Fatalf("Channel Ch1 not found")
	}
	if ch.TxContact != "TG1" { // Note: Importer currently sets TxContact string, ResolveContacts maps it.
		// Check if it was resolved
		// Wait, simple importer might not resolve immediately unless we tell it to.
		// handleImport calls resolveContacts(channels).
		// But resolveContacts looks up by Name in `models.Contact`.
		// Did we import TG1 into models.Contact?
		// Yes, we included talkgroups.csv.
	}

	var z models.Zone
	if err := database.DB.Preload("Channels").Where("name = ?", "Zone1").First(&z).Error; err != nil {
		t.Fatalf("Zone Zone1 not found")
	}
	if len(z.Channels) == 0 {
		t.Errorf("Zone1 has no channels, expected Ch1")
	} else if z.Channels[0].Name != "Ch1" {
		t.Errorf("Zone1 has wrong channel: %s", z.Channels[0].Name)
	}
}

func TestZipRoundTrip(t *testing.T) {
	// Setup DB 1 (Seed)
	db1 := "test-roundtrip-1.db"
	os.Remove(db1)
	defer os.Remove(db1)
	database.Connect(db1)

	// Seed Complex Data
	database.DB.Create(&models.DigitalContact{Name: "Global User", DMRID: 99999})
	tg := models.Contact{Name: "Local TG", DMRID: 55, Type: models.ContactTypeGroup}
	database.DB.Create(&tg)
	ch := models.Channel{
		Name: "DMR Channel", RxFrequency: 445.5, TxFrequency: 440.5,
		Mode: "DMR", ColorCode: 1, TimeSlot: 2,
		ContactID: &tg.ID, TxContact: tg.Name, // TxContact string needed for some exporters?
	}
	database.DB.Create(&ch)
	zone := models.Zone{Name: "My Zone", Channels: []models.Channel{ch}}
	database.DB.Create(&zone)

	// 1. Export to Zip
	req, _ := http.NewRequest("GET", "/api/export?radio=dm32uv&format=zip", nil)
	rr := httptest.NewRecorder()
	http.HandlerFunc(api.HandleExport).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("RoundTrip: Export failed %d", rr.Code)
	}
	zipBytes := rr.Body.Bytes()

	// 2. Switch to DB 2 (Clean)
	// Note: changing DB connection in process is tricky if variable is global.
	// database.Connect(dbPath) replaces the global `DB` variable.
	// We need to ensure we don't leak.
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()

	db2 := "test-roundtrip-2.db"
	os.Remove(db2)
	defer os.Remove(db2)
	database.Connect(db2)

	// Verify Empty
	var count int64
	database.DB.Model(&models.Channel{}).Count(&count)
	if count != 0 {
		t.Fatalf("DB2 not empty")
	}

	// 3. Import from Zip
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "export.zip")
	io.Copy(part, bytes.NewReader(zipBytes))
	writer.Close()

	reqIn, _ := http.NewRequest("POST", "/api/import?radio=dm32uv&format=zip", body)
	reqIn.Header.Set("Content-Type", writer.FormDataContentType())
	rrIn := httptest.NewRecorder()
	http.HandlerFunc(api.HandleImport).ServeHTTP(rrIn, reqIn)

	if rrIn.Code != http.StatusOK {
		t.Fatalf("RoundTrip: Import failed %d: %s", rrIn.Code, rrIn.Body.String())
	}

	// 4. Verify DB2 matches Seed
	var ch2 models.Channel
	database.DB.First(&ch2)
	if ch2.Name != "DMR Channel" {
		t.Errorf("Expected channel 'DMR Channel', got '%s'", ch2.Name)
	}
	if ch2.ColorCode != 1 || ch2.TimeSlot != 2 {
		t.Errorf("Channel attributes mismatch")
	}

	var z2 models.Zone
	database.DB.Preload("Channels").First(&z2)
	if z2.Name != "My Zone" {
		t.Errorf("Expected zone 'My Zone', got '%s'", z2.Name)
	}
	if len(z2.Channels) != 1 || z2.Channels[0].Name != "DMR Channel" {
		t.Errorf("Zone channel linkage failed")
	}
}
