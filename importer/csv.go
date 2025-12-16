package importer

import (
	"encoding/csv"
	"fmt"
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
				switch duplex {
				case "+":
					channel.TxFrequency = channel.RxFrequency + offset
				case "-":
					channel.TxFrequency = channel.RxFrequency - offset
				default:
					channel.TxFrequency = channel.RxFrequency
				}
				// Fix floating point precision issues (e.g. 147.79999999999998)
				// Round to 6 decimal places standard in radio
				channel.TxFrequency, _ = strconv.ParseFloat(fmt.Sprintf("%.6f", channel.TxFrequency), 64)
			} else {
				channel.TxFrequency = channel.RxFrequency
			}
		}

		// Ensure RxFrequency is also rounded just in case
		channel.RxFrequency, _ = strconv.ParseFloat(fmt.Sprintf("%.6f", channel.RxFrequency), 64)

		channel.Mode = getVal("CH mode")
		if channel.Mode == "" {
			channel.Mode = getVal("Mode")
		}
		// Optimize Mode Mapping for Type/Protocol/Bandwidth
		switch channel.Mode {
		case "", "Analog", "FM":
			channel.Mode = "FM"
			channel.Type = models.ChannelTypeAnalog
			channel.Protocol = models.ProtocolFM
			channel.Bandwidth = "25"
		case "NFM":
			channel.Mode = "FM" // Internal standardized mode
			channel.Type = models.ChannelTypeAnalog
			channel.Protocol = models.ProtocolFM
			channel.Bandwidth = "12.5"
		case "AM":
			channel.Type = models.ChannelTypeAnalog
			channel.Protocol = models.ProtocolAM
			channel.Bandwidth = "25"
			channel.Bandwidth = "25"
		case "DMR":
			channel.Mode = "DMR"
			channel.Type = models.ChannelTypeDigitalDMR
			channel.Protocol = models.ProtocolDMR
			channel.Bandwidth = "12.5"
		case "DN":
			channel.Mode = "DN"
			channel.Type = models.ChannelTypeDigitalYSF
			channel.Protocol = models.ProtocolFusion
			channel.Bandwidth = "12.5"
		case "DV":
			channel.Mode = "DV"
			channel.Type = models.ChannelTypeDigitalDStar
			channel.Protocol = models.ProtocolDStar
			channel.Bandwidth = "12.5"
		case "P25":
			channel.Mode = "P25"
			channel.Type = models.ChannelTypeDigitalP25
			channel.Protocol = "P25"
			channel.Bandwidth = "12.5"
		case "Digital": // Generic Digital fallback logic
			channel.Mode = "DMR" // Assume DMR logic for generic "Digital"
			channel.Type = models.ChannelTypeDigitalDMR
			channel.Protocol = models.ProtocolDMR
			channel.Bandwidth = "12.5"
		default:
			// Fallback for unknown modes
			channel.Type = models.ChannelTypeAnalog
			channel.Protocol = models.ProtocolFM
			channel.Bandwidth = "25"
		}

		// Explicit Bandwidth override if column exists
		if bw := getVal("Bandwidth"); bw != "" {
			channel.Bandwidth = bw
		}

		// Power Mapping
		powerStr := getVal("Power")
		if powerStr != "" {
			// Try to parse wattage logic similar to Chirp
			cleanPowerStr := powerStr
			if len(powerStr) > 0 && (powerStr[len(powerStr)-1] == 'W' || powerStr[len(powerStr)-1] == 'w') {
				cleanPowerStr = powerStr[:len(powerStr)-1]
			}
			if watts, err := strconv.ParseFloat(cleanPowerStr, 64); err == nil {
				if watts > 25 {
					channel.Power = "High"
				} else if watts > 5 {
					channel.Power = "Mid"
				} else {
					channel.Power = "Low"
				}
			} else {
				// Direct match check
				if powerStr == "High" || powerStr == "Mid" || powerStr == "Low" {
					channel.Power = powerStr
				} else {
					// Default or leave as is? Model expects specific strings.
					// If unknown string, maybe default to High or map loosely.
					channel.Power = "High"
				}
			}
		} else {
			// Check legacy/other columns for Power if needed?
			// For now, if missing, default validation might handle it or it stays empty (defaulting happening in UI/Export)
			// But user wants it set.
			if channel.Power == "" {
				channel.Power = "High" // Default
			}
		}

		// DMR specific
		if channel.Mode == "DMR" {
			ccStr := getVal("RX CC") // Assuming RX CC is the main one
			if ccStr != "" {
				channel.ColorCode, _ = strconv.Atoi(ccStr)
			}
			tsStr := getVal("RX TS") // "Slot 1" or "Slot 2"
			switch tsStr {
			case "Slot 1":
				channel.TimeSlot = 1
			case "Slot 2":
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
			switch toneMode {
			case "Tone":
				channel.TxTone = getVal("rToneFreq") // In Chirp, rToneFreq is the TX Tone for "Tone" mode
			case "TSQL":
				channel.TxTone = getVal("cToneFreq")
				channel.RxTone = getVal("cToneFreq")
			case "DTCS":
				channel.SquelchType = "DCS"
				channel.TxDCS = getVal("DtcsCode")
				channel.RxDCS = getVal("DtcsCode")
				// Chirp has DtcsPolarity "NN", "RN", etc. to distinguish invert. Ignoring for now.
			case "Cross":
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

		// Default SquelchType if missing
		if channel.SquelchType == "" {
			channel.SquelchType = "None"
		}

		channels = append(channels, channel)
	}

	return channels, nil
}

// ImportGenericTalkgroups imports contacts (talkgroups) from a simple CSV (Name,ID,Type).
func ImportGenericTalkgroups(r io.Reader) ([]models.Contact, error) {
	csvReader := csv.NewReader(r)
	csvReader.LazyQuotes = true
	headers, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[h] = i
	}

	var contacts []models.Contact

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		getVal := func(colName string) string {
			if idx, ok := headerMap[colName]; ok && idx < len(record) {
				return record[idx]
			}
			return ""
		}

		// Try to match common headers
		name := getVal("Name")
		if name == "" {
			name = getVal("Talkgroup")
		}

		idStr := getVal("ID")
		if idStr == "" {
			idStr = getVal("DMRID")
		}

		if name == "" || idStr == "" {
			continue
		}

		id, _ := strconv.Atoi(idStr)
		c := models.Contact{
			Name:  name,
			DMRID: id,
			Type:  models.ContactTypeGroup, // Default
		}

		typeStr := getVal("Type")
		if typeStr != "" {
			if typeStr == "Private" || typeStr == "Private Call" {
				c.Type = models.ContactTypePrivate
			} else if typeStr == "All" || typeStr == "All Call" {
				c.Type = models.ContactTypeAllCall
			}
		}

		contacts = append(contacts, c)
	}

	return contacts, nil
}
