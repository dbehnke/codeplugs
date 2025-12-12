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
		t.Errorf("Contact should not be soft deleted (DeletedAt should be invalid/null), got Valid=true")
	}
	if resurrected.Callsign != "GHOST" {
		t.Errorf("Expected Updated Callsign GHOST, got %s", resurrected.Callsign)
	}
}
