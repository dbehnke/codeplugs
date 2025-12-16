package main

import (
	"bytes"
	"codeplugs/database"
	"codeplugs/models"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestImportResurrectsSoftDeleted(t *testing.T) {
	tmpDB, _ := os.CreateTemp("", "test-resurrect-*.db")
	defer os.Remove(tmpDB.Name())
	database.Connect(tmpDB.Name())

	// 1. Create a Contact
	contact := models.DigitalContact{
		Name:     "Ghost Contact",
		DMRID:    666,
		Callsign: "GHOST",
	}
	database.DB.Create(&contact)

	// 2. Soft Delete it
	database.DB.Delete(&contact)

	// Verify it's gone (gorm default scope)
	var count int64
	database.DB.Model(&models.DigitalContact{}).Where("dmr_id = ?", 666).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 contacts after delete, got %d", count)
	}

	// 3. Import CSV containing the SAME contact
	radioCSV := `radio_id,callsign,first_name,last_name
666,GHOST,Ghost,Contact
`
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("format", "radioid")
	part, _ := writer.CreateFormFile("file", "radioid.csv")
	part.Write([]byte(radioCSV))
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	http.HandlerFunc(handleImport).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Import failed with status: %d", rr.Code)
	}

	// 4. Verify Contact is Resurrected
	var resurrected models.DigitalContact
	if err := database.DB.Where("dmr_id = ?", 666).First(&resurrected).Error; err != nil {
		t.Fatalf("Failed to find resurrected contact: %v", err)
	}
	if resurrected.DeletedAt.Valid {
		t.Error("Contact should be active (DeletedAt invalid), but it is valid (deleted)")
	}
}

func TestImportGenericFallback(t *testing.T) {
	// Setup DB
	tmpDB, _ := os.CreateTemp("", "test-fallback-*.db")
	defer os.Remove(tmpDB.Name())
	database.Connect(tmpDB.Name())
	defer database.Close()

	// Minimal CSV data WITHOUT Location/CrossMode to force Generic Importer
	// But WITH Power and NFM to test the new logic in Generic importer
	// Also use imprecise freq to test rounding
	csvData := `Name,Frequency,Mode,Power
GenericFallbackCh,146.520000000000001,NFM,50W
`
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test_fallback.csv")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte(csvData))
	writer.WriteField("format", "generic")
	writer.Close()

	req, err := http.NewRequest("POST", "/api/import", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	http.HandlerFunc(handleImport).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify DB content
	var ch models.Channel
	database.DB.Where("name = ?", "GenericFallbackCh").First(&ch)
	if ch.ID == 0 {
		t.Fatal("Channel GenericFallbackCh not found in DB")
	}
	// Check Power (should be "High" from 50W mapping in Generic importer now)
	if ch.Power != "High" {
		t.Errorf("Expected Power 'High', got '%s'", ch.Power)
	}
	// Check Bandwidth/Type default for NFM (should be 12.5)
	if ch.Bandwidth != "12.5" {
		t.Errorf("Expected Bandwidth '12.5', got '%s'", ch.Bandwidth)
	}
	if ch.Type != models.ChannelTypeAnalog {
		t.Errorf("Expected Type 'Analog', got '%s'", ch.Type)
	}
	// Check SquelchType default
	if ch.SquelchType != "None" {
		t.Errorf("Expected SquelchType 'None', got '%s'", ch.SquelchType)
	}
	// Check Frequency Rounding
	if ch.RxFrequency != 146.52 {
		t.Errorf("Expected RxFrequency 146.52, got %v", ch.RxFrequency)
	}
}
