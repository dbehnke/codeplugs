package exporter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"codeplugs/models"
)

// ExportDB25D exports channels to a CSV file in the DB25-D format.
func ExportDB25D(channels []models.Channel, w io.Writer, useFirstName bool) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write Header
	header := []string{
		"Z-4", "CH mode", "CH Name", "RX Freq", "TX Freq", "Power", "RX Only", "Alarm ACK", "Prompt", "PCT",
		"RX TS", "TX TS", "RX CC", "TX CC", "Msg Type", "TX Policy", "RX Group", "Encryption List",
		"Scan List", "Contacts", "EAS", "Relay Monitor", "Relay mode", "Bandwidth", "RX QT/DQT", "TX QT/DQT", "APRS",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for i, ch := range channels {
		record := make([]string, len(header))
		record[0] = strconv.Itoa(i + 1) // No. (or Z-4 index?)

		// CH mode
		if ch.Mode == "DMR" {
			record[1] = "Digital"
		} else {
			record[1] = "Analog" // Or "FM"? Sample said "Digital"
		}

		record[2] = ch.Name
		record[3] = fmt.Sprintf("%.5f", ch.RxFrequency)
		record[4] = fmt.Sprintf("%.5f", ch.TxFrequency)
		record[5] = "High" // Default or from DB if available
		if ch.Power != "" {
			record[5] = ch.Power
		}
		record[6] = "Off"   // RX Only
		record[7] = "Off"   // Alarm ACK
		record[8] = "Off"   // Prompt
		record[9] = "Patcs" // PCT - from sample

		// DMR Fields
		if ch.Mode == "DMR" {
			record[10] = fmt.Sprintf("Slot %d", ch.TimeSlot)
			record[11] = fmt.Sprintf("Slot %d", ch.TimeSlot)
			record[12] = strconv.Itoa(ch.ColorCode)
			record[13] = strconv.Itoa(ch.ColorCode)
			record[14] = "Unconfirmed Data"
			record[15] = "Polite to CC"
			record[16] = "None" // RX Group
			if ch.RxGroup != "" {
				record[16] = ch.RxGroup
			}
			record[19] = "None" // Contacts
			if ch.Contact != nil {
				if ch.Contact.Name != "" {
					record[19] = ch.Contact.Name
				}
			} else if ch.TxContact != "" {
				record[19] = ch.TxContact
			}
		} else {
			record[10] = "Slot 1"
			record[11] = "Slot 1"
			record[12] = "1"
			record[13] = "1"
			record[14] = "Unconfirmed Data"
			record[15] = "Polite to CC"
			record[16] = "None"
			record[19] = "None"
		}

		record[17] = "Off" // Encryption List
		record[18] = "Off" // Scan List
		record[20] = "Off" // EAS
		record[21] = "Off" // Relay Monitor
		record[22] = "Off" // Relay mode

		record[23] = "12.5" // Bandwidth
		if ch.Bandwidth != "" {
			record[23] = ch.Bandwidth
		}

		// Tone
		if ch.Tone != "" {
			record[24] = "Off"   // RX QT/DQT - usually Off unless TSQ
			record[25] = ch.Tone // TX QT/DQT
		} else {
			record[24] = "Off"
			record[25] = "Off"
		}

		record[26] = "Off" // APRS

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
