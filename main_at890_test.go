package main

import (
	"archive/zip"
	"bytes"
	"codeplugs/api"
	"codeplugs/database"
	"codeplugs/models"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandleExport_AnyTone890(t *testing.T) {
	// Setup in-memory DB
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file:memdb_main_at890?mode=memory&cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	database.DB = db
	db.AutoMigrate(&models.Channel{}, &models.Contact{}, &models.Zone{}, &models.ZoneChannel{}, &models.DigitalContact{})

	// Seed Data
	db.Create(&models.Channel{Name: "TestChan", RxFrequency: 146.520, TxFrequency: 146.520, Type: models.ChannelTypeAnalog})
	db.Create(&models.Contact{Name: "TestGroup", Type: models.ContactTypeGroup, DMRID: 12345})

	// Create request
	req, err := http.NewRequest("GET", "/api/export?radio=at890", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	// Call handler
	handler := http.HandlerFunc(api.HandleExport)
	handler.ServeHTTP(rr, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check headers
	assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"))

	// Verify ZIP content
	body := rr.Body.Bytes()
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("Failed to create zip reader: %v", err)
	}

	expectedFiles := []string{"Channel.CSV", "DMRTalkGroups.CSV", "DMRZone.CSV", "DMRDigitalContactList.CSV"}
	for _, expected := range expectedFiles {
		found := false
		for _, file := range zipReader.File {
			if file.Name == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected file %s in zip", expected)
	}
}
