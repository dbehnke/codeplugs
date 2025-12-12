package importer

import (
	"strings"
	"testing"
)

func TestImportRadioIDCSV_NameSplitting(t *testing.T) {
	csvContent := `radio_id,callsign,first_name,last_name,city,state,country,remarks
1234567,W1AW,Hiram,Maxim,Newington,CT,United States,Founder
7654321,K1ABC,John,,Boston,MA,United States,No Last Name
`
	reader := strings.NewReader(csvContent)
	contacts, err := ImportRadioIDCSV(reader, nil)
	if err != nil {
		t.Fatalf("ImportRadioIDCSV failed: %v", err)
	}

	if len(contacts) != 2 {
		t.Fatalf("Expected 2 contacts, got %d", len(contacts))
	}

	c1 := contacts[0]
	// DigitalContact only stores joined Name
	if c1.Name != "Hiram Maxim" {
		t.Errorf("Expected Name 'Hiram Maxim', got '%s'", c1.Name)
	}

	c2 := contacts[1]
	if c2.Name != "John" {
		t.Errorf("Expected Name 'John', got '%s'", c2.Name)
	}
}
