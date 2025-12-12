package importer

import (
	"encoding/csv"
	"io"
	"strconv"

	"codeplugs/models"
)

// ImportGenericCSV imports channels from a CSV file with specific headers.
// This is a basic implementation that assumes headers match the struct fields or a known mapping.
// For now, let's implement a simple mapping for Repeaterbook style or a generic one.
// ImportChannelsCSV imports channels from a CSV stream.
func ImportChannelsCSV(r io.Reader) ([]models.Channel, error) {

	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true
	headers, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	// Map headers to indices
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[h] = i
	}

	var channels []models.Channel

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		channel := models.Channel{}

		// Helper to get value safely
		getVal := func(colName string) string {
			if idx, ok := headerMap[colName]; ok && idx < len(record) {
				return record[idx]
			}
			return ""
		}

		// Basic mapping logic (adjust based on actual CSV format)
		// Try DB25-D headers first
		channel.Name = getVal("CH Name")
		if channel.Name == "" {
			channel.Name = getVal("Name")
		}
		if channel.Name == "" {
			channel.Name = getVal("Callsign")
		}

		rxStr := getVal("RX Freq")
		if rxStr == "" {
			rxStr = getVal("Frequency")
		}
		if rxStr != "" {
			channel.RxFrequency, _ = strconv.ParseFloat(rxStr, 64)
		}

		txStr := getVal("TX Freq")
		if txStr == "" {
			txStr = getVal("Input Freq")
		}
		if txStr != "" {
			channel.TxFrequency, _ = strconv.ParseFloat(txStr, 64)
		} else {
			// Check for Chirp Duplex/Offset
			duplex := getVal("Duplex")
			offsetStr := getVal("Offset")
			if duplex != "" && offsetStr != "" {
				offset, _ := strconv.ParseFloat(offsetStr, 64)
				if duplex == "+" {
					channel.TxFrequency = channel.RxFrequency + offset
				} else if duplex == "-" {
					channel.TxFrequency = channel.RxFrequency - offset
				} else {
					channel.TxFrequency = channel.RxFrequency
				}
			} else {
				channel.TxFrequency = channel.RxFrequency
			}
		}

		channel.Mode = getVal("CH mode")
		if channel.Mode == "" {
			channel.Mode = getVal("Mode")
		}
		// Normalize Mode
		if channel.Mode == "Digital" {
			channel.Mode = "DMR"
		} else if channel.Mode == "NFM" {
			channel.Mode = "FM"
		}

		channel.Bandwidth = getVal("Bandwidth")

		// DMR specific
		if channel.Mode == "DMR" {
			ccStr := getVal("RX CC") // Assuming RX CC is the main one
			if ccStr != "" {
				channel.ColorCode, _ = strconv.Atoi(ccStr)
			}
			tsStr := getVal("RX TS") // "Slot 1" or "Slot 2"
			if tsStr == "Slot 1" {
				channel.TimeSlot = 1
			} else if tsStr == "Slot 2" {
				channel.TimeSlot = 2
			}

			channel.RxGroup = getVal("RX Group")
			channel.TxContact = getVal("Contacts")
		}

		// Squelch Mapping (Chirp & DB25-D)
		// 1. Try generic "Tone" field for TX Tone (DB25-D TX QT/DQT)
		// DB25-D "TX QT/DQT" could be CTCSS or DCS. Format usually "88.5" or "D023N"
		rawTone := getVal("TX QT/DQT")
		if rawTone != "" && rawTone != "Off" {
			// This logic handles simple cases. Ideally split if it starts with "D" -> DCS
			if len(rawTone) > 0 && rawTone[0] == 'D' {
				channel.SquelchType = "DCS"
				channel.TxDCS = rawTone
			} else {
				channel.SquelchType = "Tone"
				channel.TxTone = rawTone
			}
		}

		// 2. Chirp Specific Logic (Overrides above if present)
		toneMode := getVal("Tone")
		if toneMode != "" {
			channel.SquelchType = toneMode // Tone, TSQL, DTCS, Cross
			if toneMode == "Tone" {
				channel.TxTone = getVal("rToneFreq") // In Chirp, rToneFreq is the TX Tone for "Tone" mode
			} else if toneMode == "TSQL" {
				channel.TxTone = getVal("cToneFreq")
				channel.RxTone = getVal("cToneFreq")
			} else if toneMode == "DTCS" {
				channel.SquelchType = "DCS"
				channel.TxDCS = getVal("DtcsCode")
				channel.RxDCS = getVal("DtcsCode")
				// Chirp has DtcsPolarity "NN", "RN", etc. to distinguish invert. Ignoring for now.
			} else if toneMode == "Cross" {
				// Cross mode is complex in Chirp. Mapping simplified version.
				// "Cross" mode in Chirp uses CrossMode field to define:
				// "Tone->Tone", "DTCS->", "->DTCS", "Tone->DTCS", etc.
				// User wanted explicit fields, so we fill them if we can map them.
				crossMode := getVal("CrossMode")
				// Examples: "Tone->Tone" (Split tones)
				if crossMode == "Tone->Tone" {
					channel.TxTone = getVal("rToneFreq")
					channel.RxTone = getVal("cToneFreq")
				}
				// Examples: "DTCS->DTCS" (Split DCS - unlikely but possible)
			}
		}

		// Backward compatibility: Populate old Tone field for now if needed, or leave it.
		// Setting old Tone to TxTone for display consistency until UI fully updated
		if channel.TxTone != "" {
			channel.Tone = channel.TxTone
		} else if channel.TxDCS != "" {
			channel.Tone = channel.TxDCS
		}

		channels = append(channels, channel)
	}

	return channels, nil
}
