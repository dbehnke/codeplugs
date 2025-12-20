package main

import (
	"bytes"
	"codeplugs/api"
	"codeplugs/database"
	"codeplugs/models"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Start hub for tests to avoid deadlock on broadcast
	go api.Hub.Run()
	os.Exit(m.Run())
}

func TestHandleContacts(t *testing.T) {
	// Setup temporary DB
	tmpDB, _ := os.CreateTemp("", "test-contacts-*.db")
	database.Connect(tmpDB.Name())
	defer os.Remove(tmpDB.Name())

	// AutoMigrate is called in Connect in the real app, but let's ensure it here or rely on Connect
	// database.Connect calls AutoMigrate for Channel and Contact now.

	// 1. Test GET empty
	req, _ := http.NewRequest("GET", "/api/contacts", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.HandleContacts)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("GET /api/contacts returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &wrapper); err != nil {
		t.Errorf("Failed to parse wrapper: %v", err)
	}

	// "data" field in wrapper maps to map[string]interface{"data": []} because RespondJSON wraps the map returned by handler.
	// api.HandleContacts returns map[string]interface{}{"data": contacts}

	// So RespondJSON output: { "success": true, "data": { "data": [] } }
	// This double nesting of "data" is what complicates things.

	var inner map[string][]models.Contact
	if err := json.Unmarshal(wrapper.Data, &inner); err != nil {
		t.Errorf("Failed to parse inner data: %v", err)
	}

	if len(inner["data"]) != 0 {
		t.Errorf("Expected 0 contacts, got %d", len(inner["data"]))
	}

	// 2. Test POST valid contact
	newContact := models.Contact{
		Name:  "Local",
		DMRID: 9,
		Type:  models.ContactTypeGroup,
	}
	body, _ := json.Marshal(newContact)
	req, _ = http.NewRequest("POST", "/api/contacts", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("POST /api/contacts returned wrong status code: got %v want %v. Body: %s", status, http.StatusOK, rr.Body.String())
	} else {
		t.Logf("POST Body: %s", rr.Body.String())
	}

	// 3. Verify it was saved
	req, _ = http.NewRequest("GET", "/api/contacts", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	bodyBytes := rr.Body.Bytes()

	json.Unmarshal(bodyBytes, &wrapper)
	json.Unmarshal(wrapper.Data, &inner)

	if len(inner["data"]) != 1 {
		t.Fatalf("Expected 1 contact, got %d. Body: %s", len(inner["data"]), string(bodyBytes))
	}
	if inner["data"][0].Name != "Local" {
		t.Errorf("Expected contact name Local, got %s", inner["data"][0].Name)
	}
}

func TestContactUniqueness(t *testing.T) {
	// Setup DB
	tmpDB, _ := os.CreateTemp("", "test-*.db")
	defer os.Remove(tmpDB.Name())
	database.Connect(tmpDB.Name())

	// Create first contact
	c1 := models.Contact{Name: "Group 1", DMRID: 100, Type: models.ContactTypeGroup}
	if err := database.DB.Create(&c1).Error; err != nil {
		t.Fatalf("Failed to create first contact: %v", err)
	}

	// Create duplicate contact (Same ID, Same Type) - Should Fail
	c2 := models.Contact{Name: "Group 1 Duplicate", DMRID: 100, Type: models.ContactTypeGroup}
	if err := database.DB.Create(&c2).Error; err == nil {
		t.Errorf("Expected error when creating duplicate contact (ID 100, Group), got nil")
	}

	// Create different contact (Same ID, Different Type) - Should Pass
	c3 := models.Contact{Name: "Private 1", DMRID: 100, Type: models.ContactTypePrivate}
	if err := database.DB.Create(&c3).Error; err != nil {
		t.Errorf("Expected success when creating contact with same ID but different type, got error: %v", err)
	}
}

func TestContactReferentialIntegrity(t *testing.T) {
	tmpDB, _ := os.CreateTemp("", "test-integrity-*.db")
	defer os.Remove(tmpDB.Name())
	database.Connect(tmpDB.Name())

	// 1. Create Contact
	contact := models.Contact{Name: "Used Contact", DMRID: 999, Type: models.ContactTypeGroup}
	database.DB.Create(&contact)

	// 2. Create Channel using Contact
	channel := models.Channel{
		Name:        "Test Channel",
		RxFrequency: 146.52,
		TxFrequency: 146.52,
		ContactID:   &contact.ID,
	}
	database.DB.Create(&channel)

	// 3. Attempt Delete Contact (Should Fail)
	req, _ := http.NewRequest("DELETE", "/api/contacts?id="+jsonNumber(contact.ID), nil)
	rr := httptest.NewRecorder()
	http.HandlerFunc(api.HandleContacts).ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("Expected 409 Conflict when deleting used contact, got %d", rr.Code)
	}
}

func jsonNumber(id uint) string {
	b, _ := json.Marshal(id)
	return string(b)
}

func TestImportResolvesContactName(t *testing.T) {
	tmpDB, _ := os.CreateTemp("", "test-import-*.db")
	defer os.Remove(tmpDB.Name())
	database.Connect(tmpDB.Name())

	// 1. Create Contact
	contact := models.Contact{Name: "Biff", DMRID: 888, Type: models.ContactTypeGroup}
	database.DB.Create(&contact)

	// 2. Create CSV File
	csvContent := `Name,RX Freq,Mode,Contacts
MyChannel,446.000,DMR,Biff
`
	tmpCSV, _ := os.CreateTemp("", "test-import-*.csv")
	defer os.Remove(tmpCSV.Name())
	tmpCSV.WriteString(csvContent)
	tmpCSV.Close()

	// 3. Call HandleImport (mocking request)
	// We need to construct a multipart request
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", tmpCSV.Name())
	io.Copy(part, bytes.NewReader([]byte(csvContent))) // Write content directly to form part
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	http.HandlerFunc(api.HandleImport).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Import failed with status: %d", rr.Code)
	}

	// 4. Verify Channel Linked to Contact
	var ch models.Channel
	if err := database.DB.Where("name = ?", "MyChannel").First(&ch).Error; err != nil {
		t.Fatalf("Channel not found: %v", err)
	}

	if ch.ContactID == nil {
		t.Fatal("ContactID is nil, expected link to 'Biff'")
	}
	if *ch.ContactID != contact.ID {
		t.Errorf("ContactID mismatch. Got %d, want %d", *ch.ContactID, contact.ID)
	}
}

func TestImportRadioIDIntegration(t *testing.T) {
	tmpDB, _ := os.CreateTemp("", "test-radioid-*.db")
	defer os.Remove(tmpDB.Name())
	database.Connect(tmpDB.Name())

	// CSV Content
	radioCSV := `radio_id,callsign,first_name,last_name
111,N0ONE,No,One
222,N0TWO,No,Two
`
	// Create request body
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Add format
	writer.WriteField("format", "radioid")

	// Add file
	part, _ := writer.CreateFormFile("file", "radioid.csv")
	part.Write([]byte(radioCSV))

	writer.Close()

	req, _ := http.NewRequest("POST", "/api/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	http.HandlerFunc(api.HandleImport).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Import failed with status: %d", rr.Code)
	}

	// Verify DigitalContacts
	var contacts []models.DigitalContact
	database.DB.Find(&contacts)

	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(contacts))
	}
	if len(contacts) > 0 {
		// DigitalContact doesn't have Source field, it is the source.
		// Check Callsign (order might vary, but let's check one)
		found := false
		for _, c := range contacts {
			if c.Callsign == "N0ONE" && c.Name == "No One" {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected to find contact N0ONE (No One), but didn't")
		}
	}
}
