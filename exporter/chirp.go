package exporter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"codeplugs/models"
)

// ExportChirpCSV exports channels to a Chirp-formatted CSV file.
func ExportChirpCSV(channels []models.Channel, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Chirp Header
	header := []string{
		"Location", "Name", "Frequency", "Duplex", "Offset", "Tone", "rToneFreq", "cToneFreq", "DtcsCode", "DtcsPolarity", "RxDtcsCode", "CrossMode", "Mode", "TStep", "Skip", "Power", "Comment", "URCALL", "RPT1CALL", "RPT2CALL", "DVCODE",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for i, ch := range channels {
		// Filter out Digital channels for Chirp
		if ch.Mode == "DMR" {
			continue
		}

		record := make([]string, len(header))
		record[0] = strconv.Itoa(i + 1) // Location
		record[1] = ch.Name
		record[2] = fmt.Sprintf("%.6f", ch.RxFrequency)

		// Duplex & Offset logic
		diff := ch.TxFrequency - ch.RxFrequency
		// Floating point comparison tolerance
		if diff > 0.000001 {
			record[3] = "+"
			record[4] = fmt.Sprintf("%.6f", diff)
		} else if diff < -0.000001 {
			record[3] = "-"
			record[4] = fmt.Sprintf("%.6f", -diff)
		} else {
			record[3] = "" // Simplex
			record[4] = "0.000000"
		}

		// Tone Logic
		// Default values
		record[5] = "" // Tone Mode
		record[6] = "88.5"
		record[7] = "88.5"
		record[8] = "023"
		record[9] = "NN"
		record[10] = "" // RxDtcsCode
		record[11] = "" // CrossMode

		switch ch.SquelchType {
		case "Tone":
			record[5] = "Tone"
			if ch.TxTone != "" {
				record[6] = ch.TxTone // rToneFreq
			}
		case "TSQL":
			record[5] = "TSQL"
			if ch.TxTone != "" {
				record[7] = ch.TxTone // cToneFreq
				record[6] = ch.TxTone // Set both for consistency
			}
		case "DCS":
			record[5] = "DTCS"
			if ch.TxDCS != "" {
				record[8] = ch.TxDCS // DtcsCode
			}
			if ch.RxDCS != "" {
				record[10] = ch.RxDCS
			} else {
				record[10] = ch.TxDCS // Assume symmetric if Rx missing
			}
		case "Cross":
			record[5] = "Cross"
			// Try to determine cross mode type
			if ch.TxTone != "" && ch.RxTone != "" {
				record[11] = "Tone->Tone"
				record[6] = ch.TxTone
				record[7] = ch.RxTone
			} else if ch.TxDCS != "" && ch.RxDCS != "" {
				record[11] = "Dtcs->Dtcs"
				record[8] = ch.TxDCS
				record[10] = ch.RxDCS
			} else if ch.TxTone != "" {
				// Tone -> Sql?
				record[11] = "Tone->Sql"
				record[6] = ch.TxTone
			} else {
				// Fallback
				record[11] = "Tone->Tone"
			}
		}

		// Fallback for compatibility if SquelchType is empty but Tone field is set
		if record[5] == "" && ch.Tone != "" {
			if len(ch.Tone) > 0 && ch.Tone[0] == 'D' {
				record[5] = "DTCS"
				// Strip "D" if present
				code := ch.Tone
				if len(code) > 1 && code[0] == 'D' {
					code = code[1:]
				}
				record[8] = code
				record[10] = code
			} else {
				record[5] = "TSQL"
				record[7] = ch.Tone
				record[6] = ch.Tone
			}
		}

		record[12] = ch.Mode
		if record[12] == "FM" {
			record[12] = "NFM" // Chirp usually likes NFM
		}

		record[13] = "5.00" // TStep default
		record[14] = ""     // Skip

		// Power Mapping
		// Map "High" -> "50W", "Mid" -> "25W", "Low" -> "5W"
		switch ch.Power {
		case "High":
			record[15] = "50W"
		case "Mid":
			record[15] = "25W"
		case "Low":
			record[15] = "5W"
		default:
			if ch.Power != "" {
				record[15] = ch.Power
			} else {
				record[15] = "50W" // Default
			}
		}

		record[16] = ch.Notes

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
