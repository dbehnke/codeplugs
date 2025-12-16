package importer

import (
	"codeplugs/models"
	"strings"
	"testing"
)

func TestImportChirpCSV_GenericFormat(t *testing.T) {
	// Sample data mimicking references/chirp-repeaterbook-generic-macomb.csv
	// Covering:
	// 1. High Power (50W), TSQL, NFM (12.5)
	// 2. Mid Power (approximated), Tone, FM (25)
	// 3. Low Power, DTCS, AM (25)
	// 4. CrossMode, DMR (12.5)

	csvData := `Location,Name,Frequency,Duplex,Offset,Tone,rToneFreq,cToneFreq,DtcsCode,DtcsPolarity,RxDtcsCode,CrossMode,Mode,TStep,Skip,Power,Comment
0,HighTSQL,443.625000,+,5.000000,TSQL,88.5,151.4,023,NN,023,,NFM,5.00,,50W,"High Power TSQL"
1,MidTone,147.080000,+,0.600000,Tone,100.0,88.5,023,NN,023,,FM,5.00,,10W,"Mid Power Tone"
2,LowDTCS,118.000000,+,0.000000,DTCS,88.5,123.0,023,NN,023,,AM,5.00,,5W,"Low Power AM"
3,CrossCh,443.550000,+,5.000000,Cross,107.2,88.5,023,NN,023,Tone->Tone,DMR,5.00,,50W,"Cross Mode Tone->Tone"
`

	r := strings.NewReader(csvData)
	channels, err := ImportChirpCSV(r)
	if err != nil {
		t.Fatalf("ImportChirpCSV failed: %v", err)
	}

	if len(channels) != 4 {
		t.Fatalf("Expected 4 channels, got %d", len(channels))
	}

	// 1. High Power TSQL (NFM)
	c1 := channels[0]
	if c1.Name != "HighTSQL" {
		t.Errorf("Channel 1 Name mismatch: got %s", c1.Name)
	}
	if c1.Power != "High" {
		t.Errorf("Channel 1 Power mismatch: expected High, got %s", c1.Power)
	}
	if c1.SquelchType != "TSQL" {
		t.Errorf("Channel 1 SquelchType mismatch: expected TSQL, got %s", c1.SquelchType)
	}
	if c1.Bandwidth != "12.5" {
		t.Errorf("Channel 1 Bandwidth mismatch (NFM): expected 12.5, got %s", c1.Bandwidth)
	}
	if c1.Protocol != models.ProtocolFM {
		t.Errorf("Channel 1 Protocol mismatch: expected FM, got %s", c1.Protocol)
	}

	// 2. Mid Power Tone (FM)
	c2 := channels[1]
	if c2.Power != "Mid" {
		t.Errorf("Channel 2 Power mismatch: expected Mid, got %s", c2.Power)
	}
	if c2.Bandwidth != "25" {
		t.Errorf("Channel 2 Bandwidth mismatch (FM): expected 25, got %s", c2.Bandwidth)
	}

	// 3. Low Power DTCS (AM)
	c3 := channels[2]
	if c3.Power != "Low" {
		t.Errorf("Channel 3 Power mismatch: expected Low, got %s", c3.Power)
	}
	if c3.Protocol != "AM" {
		t.Errorf("Channel 3 Protocol mismatch: expected AM, got %s", c3.Protocol)
	}
	if c3.Bandwidth != "25" {
		t.Errorf("Channel 3 Bandwidth mismatch (AM): expected 25, got %s", c3.Bandwidth)
	}

	// 4. Cross Mode Tone->Tone (DMR)
	c4 := channels[3]
	if c4.Protocol != models.ProtocolDMR {
		t.Errorf("Channel 4 Protocol mismatch: expected DMR, got %s", c4.Protocol)
	}
	if c4.Type != models.ChannelTypeDigitalDMR {
		t.Errorf("Channel 4 Type mismatch: expected Digital (DMR), got %s", c4.Type)
	}
	if c4.Bandwidth != "12.5" {
		t.Errorf("Channel 4 Bandwidth mismatch (DMR): expected 12.5, got %s", c4.Bandwidth)
	}
	// "Tone->Tone" with Rx Tone present maps to TSQL now
	if c4.SquelchType != "TSQL" {
		t.Errorf("Channel 4 SquelchType mismatch: expected TSQL, got %s", c4.SquelchType)
	}
	if c4.TxTone != "107.2" { // rToneFreq
		t.Errorf("Channel 4 TxTone mismatch: expected 107.2, got %s", c4.TxTone)
	}
	if c4.RxTone != "88.5" { // cToneFreq
		t.Errorf("Channel 4 RxTone mismatch: expected 88.5, got %s", c4.RxTone)
	}
}
