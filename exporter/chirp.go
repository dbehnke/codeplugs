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
		"Location", "Name", "Frequency", "Duplex", "Offset", "Tone", "rToneFreq", "cToneFreq", "DtcsCode", "DtcsPolarity", "Mode", "TStep", "Skip", "Comment", "URCALL", "RPT1CALL", "RPT2CALL", "DVCODE",
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
			}
		case "DCS":
			record[5] = "DTCS"
			if ch.TxDCS != "" {
				record[8] = ch.TxDCS // DtcsCode
			}
		case "Cross":
			// Simplified cross mode handling
			record[5] = "Cross"
			// Logic depends on Rx/Tx combo. Chirp uses CrossMode field to define
			// defaulting to Tone->Tone if we have split tones
			if ch.TxTone != "" && ch.RxTone != "" {
				record[15] = "Tone->Tone" // CrossMode
				record[6] = ch.TxTone
				record[7] = ch.RxTone
			}
		}

		// Fallback for compatibility if SquelchType is empty but Tone field is set
		if record[5] == "" && ch.Tone != "" {
			if len(ch.Tone) > 0 && ch.Tone[0] == 'D' {
				record[5] = "DTCS"
				record[8] = ch.Tone
			} else {
				record[5] = "TSQL"
				record[7] = ch.Tone
			}
		}

		record[10] = ch.Mode
		if record[10] == "FM" {
			record[10] = "NFM" // Chirp usually likes NFM
		}

		record[11] = "5.00" // TStep default
		record[12] = ""     // Skip
		record[13] = ch.Notes

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
