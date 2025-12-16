package exporter

import (
	"bytes"
	"codeplugs/models"
	"encoding/csv"
	"testing"
)

func TestExportChirpCSV_GenericFormat(t *testing.T) {
	channels := []models.Channel{
		{
			Name:        "HighTSQL",
			Mode:        "FM",
			RxFrequency: 146.52,
			TxFrequency: 146.52,
			Power:       "High",
			SquelchType: "TSQL",
			TxTone:      "100.0",
		},
		{
			Name:        "MidTone",
			Mode:        "FM",
			RxFrequency: 145.50,
			TxFrequency: 145.50,
			Power:       "Mid",
			SquelchType: "Tone",
			TxTone:      "88.5",
		},
		{
			Name:        "LowDTCS",
			Mode:        "FM",
			RxFrequency: 442.225,
			TxFrequency: 447.225,
			Power:       "Low",
			SquelchType: "DCS",
			TxDCS:       "023",
			RxDCS:       "023",
		},
		{
			Name:        "CrossCh",
			Mode:        "FM",
			RxFrequency: 443.55,
			TxFrequency: 448.55,
			Power:       "High",
			SquelchType: "Cross",
			TxTone:      "107.2",
			RxTone:      "88.5",
		},
	}

	buf := new(bytes.Buffer)
	err := ExportChirpCSV(channels, buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	reader := csv.NewReader(buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Read CSV failed: %v", err)
	}

	// Header + 4 channels
	if len(records) != 5 {
		t.Fatalf("Expected 5 records, got %d", len(records))
	}

	header := records[0]
	// Verify critical new headers are present
	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	requiredCols := []string{"Power", "RxDtcsCode", "CrossMode"}
	for _, col := range requiredCols {
		if _, ok := headerMap[col]; !ok {
			t.Errorf("Missing required header column: %s", col)
		}
	}

	// Check HighTSQL
	r1 := records[1]
	if r1[headerMap["Name"]] != "HighTSQL" {
		t.Errorf("R1 Name mismatch")
	}
	if r1[headerMap["Power"]] != "50W" {
		t.Errorf("R1 Power mismatch: expected 50W, got %s", r1[headerMap["Power"]])
	}
	if r1[headerMap["Tone"]] != "TSQL" {
		t.Errorf("R1 Tone Mode mismatch: expected TSQL, got %s", r1[headerMap["Tone"]])
	}

	// Check MidTone
	r2 := records[2]
	if r2[headerMap["Power"]] != "25W" {
		t.Errorf("R2 Power mismatch: expected 25W, got %s", r2[headerMap["Power"]])
	}

	// Check LowDTCS
	r3 := records[3]
	if r3[headerMap["Power"]] != "5W" {
		t.Errorf("R3 Power mismatch: expected 5W, got %s", r3[headerMap["Power"]])
	}
	if r3[headerMap["RxDtcsCode"]] != "023" {
		t.Errorf("R3 RxDtcsCode mismatch: expected 023, got %s", r3[headerMap["RxDtcsCode"]])
	}

	// Check CrossCh
	r4 := records[4]
	if r4[headerMap["CrossMode"]] != "Tone->Tone" {
		t.Errorf("R4 CrossMode mismatch: expected Tone->Tone, got %s", r4[headerMap["CrossMode"]])
	}
	if r4[headerMap["rToneFreq"]] != "107.2" {
		t.Errorf("R4 TxTone mismatch: expected 107.2, got %s", r4[headerMap["rToneFreq"]])
	}
	if r4[headerMap["cToneFreq"]] != "88.5" {
		t.Errorf("R4 RxTone mismatch: expected 88.5, got %s", r4[headerMap["cToneFreq"]])
	}
}
