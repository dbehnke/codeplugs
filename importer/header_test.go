package importer

import (
	"strings"
	"testing"
)

func TestImportRadioIDCSV_Headers(t *testing.T) {
	// Simulator inconsistent headers
	csvContent := `radio_id,callsign,First Name,Last Name,City,State,Country,Remarks
12345,KF8S,David,Behnke,City,State,Country,Remarks`

	reader := strings.NewReader(csvContent)
	contacts, err := ImportRadioIDCSV(reader, nil)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(contacts) != 1 {
		t.Fatalf("Expected 1 contact, got %d", len(contacts))
	}

	c := contacts[0]
	if c.Name != "David Behnke" {
		t.Errorf("Expected Name 'David Behnke', got '%s'", c.Name)
	}
}
