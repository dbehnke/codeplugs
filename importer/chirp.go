package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"codeplugs/models"
)

// ImportChirpCSV imports channels from a Chirp-formatted CSV stream.
func ImportChirpCSV(r io.Reader) ([]models.Channel, error) {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true
	headers, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

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

		getVal := func(colName string) string {
			if idx, ok := headerMap[colName]; ok && idx < len(record) {
				return record[idx]
			}
			return ""
		}

		channel := models.Channel{}

		// Chirp standard fields
		// Location,Name,Frequency,Duplex,Offset,Tone,rToneFreq,cToneFreq,DtcsCode,DtcsPolarity,RxDtcsCode,CrossMode,Mode,TStep,Skip,Power,Comment,URCALL,RPT1CALL,RPT2CALL,DVCODE

		channel.Name = getVal("Name")

		freqStr := getVal("Frequency")
		if freqStr != "" {
			channel.RxFrequency, _ = strconv.ParseFloat(freqStr, 64)
		}

		// Calculate Tx Freq based on Duplex and Offset
		duplex := getVal("Duplex")
		offsetStr := getVal("Offset")
		offset, _ := strconv.ParseFloat(offsetStr, 64)

		fmt.Printf("Debug: Name=%s Duplex=%s Offset=%s OffsetVal=%f\n", channel.Name, duplex, offsetStr, offset)

		switch duplex {
		case "+":
			channel.TxFrequency = channel.RxFrequency + offset
		case "-":
			channel.TxFrequency = channel.RxFrequency - offset
		default:
			channel.TxFrequency = channel.RxFrequency // Simplex or "off"
		}

		// Fix floating point precision issues (e.g. 147.79999999999998)
		// Round to 6 decimal places standard in radio
		channel.TxFrequency, _ = strconv.ParseFloat(fmt.Sprintf("%.6f", channel.TxFrequency), 64)
		channel.RxFrequency, _ = strconv.ParseFloat(fmt.Sprintf("%.6f", channel.RxFrequency), 64)

		channel.Mode = getVal("Mode")
		// Map Mode to Type, Protocol, Bandwidth
		// Map Mode to Type, Protocol, Bandwidth
		switch channel.Mode {
		case "NFM":
			channel.Type = models.ChannelTypeAnalog
			channel.Protocol = models.ProtocolFM
			channel.Bandwidth = "12.5"
			channel.Mode = "FM"
		case "FM":
			channel.Type = models.ChannelTypeAnalog
			channel.Protocol = models.ProtocolFM
			channel.Bandwidth = "25"
		case "AM":
			channel.Type = models.ChannelTypeAnalog
			channel.Protocol = models.ProtocolAM
			channel.Bandwidth = "25"
		case "DMR":
			channel.Type = models.ChannelTypeDigitalDMR
			channel.Protocol = models.ProtocolDMR
			channel.Bandwidth = "12.5"
		case "DN":
			channel.Type = models.ChannelTypeDigitalYSF
			channel.Protocol = models.ProtocolFusion
			channel.Bandwidth = "12.5"
		case "DV":
			channel.Type = models.ChannelTypeDigitalDStar
			channel.Protocol = models.ProtocolDStar
			channel.Bandwidth = "12.5" // Typical D-Star width
		case "P25":
			channel.Type = models.ChannelTypeDigitalP25
			channel.Protocol = "P25" // Assuming ProtocolP25 exists or string match
			channel.Bandwidth = "12.5"
		default:
			// Fallback
			channel.Type = models.ChannelTypeAnalog
			channel.Protocol = models.ProtocolFM
			channel.Bandwidth = "25"
		}

		// Power Mapping
		powerStr := getVal("Power") // e.g., "50W", "5.00", "High"
		if powerStr != "" {
			// Try to parse wattage
			var watts float64
			var err error

			// Remove "W" suffix if present
			cleanPowerStr := powerStr
			if len(powerStr) > 0 && (powerStr[len(powerStr)-1] == 'W' || powerStr[len(powerStr)-1] == 'w') {
				cleanPowerStr = powerStr[:len(powerStr)-1]
			}

			watts, err = strconv.ParseFloat(cleanPowerStr, 64)
			if err == nil {
				if watts > 25 {
					channel.Power = "High"
				} else if watts > 5 {
					channel.Power = "Mid"
				} else {
					channel.Power = "Low"
				}
			} else {
				// Fallback to direct string match if not a number
				channel.Power = powerStr
			}
		}

		// Tone mapping
		toneMode := getVal("Tone") // "Tone", "TSQL", "DTCS", "Cross"
		crossMode := getVal("CrossMode")

		// First pass: extract raw values based on Chirp columns
		var candidatesRxTone, candidatesTxTone, candidatesRxDCS, candidatesTxDCS string
		// Default SquelchType to None initially
		channel.SquelchType = "None"

		switch toneMode {
		case "Tone":
			candidatesTxTone = getVal("rToneFreq")
		case "TSQL":
			candidatesTxTone = getVal("cToneFreq")
			candidatesRxTone = getVal("cToneFreq")
		case "DTCS":
			channel.SquelchType = "DCS" // Initial guess, logic below refines
			candidatesTxDCS = getVal("DtcsCode")
			candidatesRxDCS = getVal("DtcsCode")
			// Check RxDtcsCode column if available for asymmetric
			if val := getVal("RxDtcsCode"); val != "" {
				candidatesRxDCS = val
			}
		case "Cross":
			if crossMode != "" {
				switch crossMode {
				case "Tone->Tone":
					candidatesTxTone = getVal("rToneFreq")
					candidatesRxTone = getVal("cToneFreq")
				case "Tone->Sql":
					candidatesTxTone = getVal("rToneFreq")
				case "Dtcs->Dtcs":
					candidatesTxDCS = getVal("DtcsCode")
					candidatesRxDCS = getVal("RxDtcsCode")
					if candidatesRxDCS == "" {
						candidatesRxDCS = getVal("DtcsCode")
					} // Fallback
				}
			}
		}

		// Final Squelch Logic based on extracted candidates:
		// User Rule: "Tone is if only the TX has a Tone set, TSQL would happen if the RX has a Tone set."

		if candidatesRxTone != "" {
			channel.SquelchType = "TSQL"
			channel.RxTone = candidatesRxTone
			channel.TxTone = candidatesTxTone // TSQL usually implies TX tone too
		} else if candidatesTxTone != "" {
			channel.SquelchType = "Tone"
			channel.TxTone = candidatesTxTone
		} else if candidatesRxDCS != "" || candidatesTxDCS != "" {
			channel.SquelchType = "DCS"
			channel.RxDCS = candidatesRxDCS
			channel.TxDCS = candidatesTxDCS
		}

		// If explicit Type override needed (e.g. forced by DMR columns)
		// ... logic for DMR detection via ColorCode ...

		// Legacy fallback if SquelchType wasn't explicitly set by valid Tone/TSQL/DTCS/Cross
		// (Though the switch above handles all known Chirp 'Tone' column values)
		if channel.SquelchType == "" && channel.Tone != "" {
			// This block was previously using channel.Tone, but we've now mapped to TxTone/RxTone etc directly.
			// Keeping legacy check just in case 'Tone' column contained something we missed,
			// but relying on the switch above is better.
		}

		// Map 'Tone' string for display compatibility if needed, using TxTone as primary
		if channel.TxTone != "" {
			channel.Tone = channel.TxTone
		} else if channel.TxDCS != "" {
			channel.Tone = fmt.Sprintf("D%s", channel.TxDCS)
		}

		channel.Notes = getVal("Comment")

		channels = append(channels, channel)
	}

	return channels, nil
}
