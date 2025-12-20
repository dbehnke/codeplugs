package main

import (
	"archive/zip"
	"bytes"
	"codeplugs/api"
	"codeplugs/database"
	"codeplugs/models"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleImport_Zip_RoamingScanList(t *testing.T) {
	setupTestDB() // Uses the helper from main_api_test.go
	db := database.DB

	// Create some base channels for membership
	// ScanList member names need to match existing channels for association to work
	ch1 := models.Channel{Name: "ScanCh1", RxFrequency: 146.52}
	ch2 := models.Channel{Name: "ScanCh2", RxFrequency: 446.00}
	db.Create(&ch1)
	db.Create(&ch2)

	// Prepare ZIP content
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// 1. AnyTone ScanList.CSV
	f1, _ := zipWriter.Create("ScanList.CSV")
	f1.Write([]byte(`"No.","Scan List Name","Scan Channel Member"
"1","TestScanList","ScanCh1|ScanCh2"`))

	// 2. AnyTone RoamChannel.CSV
	f2, _ := zipWriter.Create("RoamChannel.CSV")
	f2.Write([]byte(`"No.","Name","Receive Frequency","Transmit Frequency","Color Code","Slot"
"1","RoamCh1","440.00000","445.00000","1","1"`))

	// 3. AnyTone RoamZone.CSV
	f3, _ := zipWriter.Create("RoamZone.CSV")
	f3.Write([]byte(`"No.","Name","Roaming Channel Member"
"1","TestRoamZone","RoamCh1"`))

	zipWriter.Close()

	// Prepare Request
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "import.zip")
	io.Copy(part, buf)
	writer.WriteField("format", "zip")
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// CALL API HANDLER
	api.HandleImport(w, req)

	// ASSERTIONS
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify ScanList
	var sl models.ScanList
	if err := db.Preload("Channels").First(&sl, "name = ?", "TestScanList").Error; err != nil {
		t.Fatal("ScanList not created")
	}
	if len(sl.Channels) != 2 {
		t.Errorf("Expected 2 channels in ScanList, got %d", len(sl.Channels))
	}

	// Verify RoamingChannel
	var rc models.RoamingChannel
	if err := db.First(&rc, "name = ?", "RoamCh1").Error; err != nil {
		t.Fatal("RoamingChannel not created")
	}
	if rc.RxFrequency != 440.0 {
		t.Errorf("Expected Rx 440.0, got %f", rc.RxFrequency)
	}

	// Verify RoamingZone
	var rz models.RoamingZone
	if err := db.Preload("Channels").First(&rz, "name = ?", "TestRoamZone").Error; err != nil {
		t.Fatal("RoamingZone not created")
	}
	if len(rz.Channels) != 1 {
		t.Errorf("Expected 1 channel in RoamingZone, got %d", len(rz.Channels))
	}
	if rz.Channels[0].Name != "RoamCh1" {
		t.Errorf("Expected channel RoamCh1 in zone, got %s", rz.Channels[0].Name)
	}
}
