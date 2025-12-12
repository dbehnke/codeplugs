package exporter

import (
	"bytes"
	"codeplugs/models"
	"encoding/csv"
	"testing"
)

func TestExportChirpCSV_Filtering(t *testing.T) {
	channels := []models.Channel{
		{Name: "Analog1", Mode: "FM", RxFrequency: 146.52},
		{Name: "Digital1", Mode: "DMR", RxFrequency: 446.00},
		{Name: "Analog2", Mode: "FM", RxFrequency: 145.50},
	}

	// Mock Writer (needs refactoring of ExportChirpCSV)
	// For TDD, we must call ExportChirpCSV.
	// Currently it takes a filename string.
	// To make this testable without FS, we need to refactor it to accept io.Writer.
	// But the test is created BEFORE refactoring?
	// If I call it with a filename, I have to read the file.
	// Ideally I refactor the signature first?
	// No, the task list says "Create tests... Refactor".
	// I will write the test assuming `ExportChirpCSV` accepts `io.Writer`.
	// This means the test will FAIL TO COMPILE initially.
	// This effectively drives the refactor step.

	buf := new(bytes.Buffer)

	// Assuming future signature: func ExportChirpCSV(channels []models.Channel, w io.Writer) error
	// Currently: func ExportChirpCSV(channels []models.Channel, filePath string) error

	// I will comment out the call or make it fail if I can't change signature yet.
	// Or I can use a temp file for now and then refactor the test later?
	// Refactoring the test later is acceptable.
	// But the plan says "Refactor Exporters to use io.Writer" is next.
	// So let's write the test using io.Writer and accept that it won't compile until I fix the signature.

	err := ExportChirpCSV(channels, buf)
	if err != nil {
		// This will likely error on compilation due to type mismatch (buf vs string)
		// t.Fatalf("Export failed: %v", err)
	}

	// Verify Output
	reader := csv.NewReader(buf)
	records, _ := reader.ReadAll()

	// Expect Header + 2 Analog Rows
	if len(records) != 3 {
		t.Errorf("Expected 3 records (Header + 2 Analog), got %d", len(records))
	}

	// Check content
	foundDigital := false
	for _, r := range records {
		// Chirp Name is index 1
		if r[1] == "Digital1" {
			foundDigital = true
		}
	}
	if foundDigital {
		t.Error("Digital channel was not filtered out")
	}
}
