package exporter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"codeplugs/models"

	"gorm.io/gorm"
)

func ExportAnyTone890(db *gorm.DB, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	f1, err := os.Create(filepath.Join(outputDir, "Channel.CSV"))
	if err != nil {
		return err
	}
	defer f1.Close()
	var channels []models.Channel
	if err := db.Find(&channels).Error; err != nil {
		return err
	}
	// Fetch contacts for lookup
	var contacts []models.Contact
	if err := db.Find(&contacts).Error; err != nil {
		return err
	}
	contactMap := make(map[string]models.Contact)
	for _, c := range contacts {
		contactMap[c.Name] = c
	}

	if err := ExportAnyTone890Channels(channels, contactMap, f1); err != nil {
		return err
	}

	f2, err := os.Create(filepath.Join(outputDir, "DMRTalkGroups.CSV"))
	if err != nil {
		return err
	}
	defer f2.Close()
	// Filter for talkgroups export
	var talkgroups []models.Contact
	for _, c := range contacts {
		if c.Type == models.ContactTypeGroup || c.Type == models.ContactTypePrivate || c.Type == models.ContactTypeAllCall {
			talkgroups = append(talkgroups, c)
		}
	}
	if err := ExportAnyTone890Talkgroups(talkgroups, f2); err != nil {
		return err
	}

	f3, err := os.Create(filepath.Join(outputDir, "DMRZone.CSV"))
	if err != nil {
		return err
	}
	defer f3.Close()
	var zones []models.Zone
	if err := db.Preload("Channels").Find(&zones).Error; err != nil {
		return err
	}
	if err := ExportAnyTone890Zones(zones, f3); err != nil {
		return err
	}

	f4, err := os.Create(filepath.Join(outputDir, "DMRDigitalContactList.CSV"))
	if err != nil {
		return err
	}
	defer f4.Close()
	var digitalContacts []models.DigitalContact
	if err := db.Find(&digitalContacts).Error; err != nil {
		return err
	}
	if err := ExportAnyTone890DigitalContacts(digitalContacts, f4); err != nil {
		return err
	}

	return nil
}

func ExportAnyTone890Channels(channels []models.Channel, contactMap map[string]models.Contact, w io.Writer) error {
	header := []string{
		"No.", "Channel Name", "Receive Frequency", "Transmit Frequency", "Channel Type", "Transmit Power", "Band Width", "CTCSS/DCS Decode", "CTCSS/DCS Encode", "Contact/Talk Group", "Contact/Talk Group Call Type", "Contact/Talk Group TG/DMR ID", "Radio ID", "Busy Lock/TX Permit", "Squelch Mode", "Optional Signal", "DTMF ID", "2Tone ID", "5Tone ID", "PTT ID", "RX Color Code", "Slot", "Scan List", "Receive Group List", "PTT Prohibit", "Reverse", "Digital Duplex", "Slot Suit", "AES Digital Encryption", "Digital Encryption", "Call Confirmation", "Talk Around(Simplex)", "Work Alone", "Custom CTCSS", "2TONE Decode", "Ranging", "Idle TX", "APRS RX", "Analog APRS PTT Mode", "Digital APRS PTT Mode", "APRS Report Type", "Digital APRS Report Channel", "Correct Frequency[Hz]", "SMS Confirmation", "Exclude channel from roaming", "DMR MODE", "DataACK Disable", "R5toneBot", "R5ToneEot", "Auto Scan", "Ana APRS Mute", "Send Talker Alias DMR/NX", "AnaAprsTxPath", "ARC4", "ex_emg_kind", "Rpga_Mdc", "DisturEn", "DisturFreq", "dmr_crc_ignore", "compand", "tx_talkalaes", "dup_call", "tx_int", "BtRxState", "idle_tx", "nxdn_wn", "NxdnRpga", "nxdnSqCon", "NxdnTxBusy", "NxDnPttId", "EnRan", "DeRan", "NxdnEncry", "NxdnGroupId", "NxdnIdNum", "NxdnStateNum", "txcc",
	}

	// Using manual writer for forced quotes
	if err := writeAnyToneRecord(w, header); err != nil {
		return err
	}

	// channels slice passed in

	for i, c := range channels {
		record := make([]string, len(header))
		record[0] = strconv.Itoa(i + 1)
		record[1] = c.Name
		record[2] = fmt.Sprintf("%.5f", c.RxFrequency)
		record[3] = fmt.Sprintf("%.5f", c.TxFrequency)
		if c.Type == models.ChannelTypeDigital {
			record[4] = "D-Digital"
		} else {
			record[4] = "A-Analog"
		}
		record[5] = c.Power
		if record[5] == "" {
			record[5] = "High"
		}
		record[6] = c.Bandwidth

		if record[6] == "" {
			record[6] = "12.5K"
		} else {
			record[6] = strings.TrimSuffix(record[6], "Hz")
		}
		record[7] = c.RxTone
		if record[7] == "" || record[7] == "None" {
			record[7] = "Off"
		}
		record[8] = c.TxTone
		if record[8] == "" || record[8] == "None" {
			record[8] = "Off"
		}
		record[9] = c.TxContact
		if record[9] == "" {
			record[9] = "None"
		}

		// Lookup Contact
		record[10] = "Group Call" // Default
		record[11] = "1"          // Default ID
		if contact, ok := contactMap[c.TxContact]; ok {
			record[11] = strconv.Itoa(contact.DMRID)
			switch contact.Type {
			case models.ContactTypePrivate:
				record[10] = "Private Call"
			case models.ContactTypeAllCall:
				record[10] = "All Call"
			default:
				record[10] = "Group Call"
			}
		}

		record[12] = "" // Radio ID (Blank in working for Analog)
		// For Digital, usually needs a value. If generic logic:
		if c.Type == models.ChannelTypeDigital && record[12] == "" {
			record[12] = "1" // Default Radio ID for Digital
		}

		record[13] = c.TxPermit // Busy Lock/TX Permit
		if record[13] == "" {
			record[13] = "Off"
		}
		record[14] = "Carrier" // Squelch Mode
		record[15] = "Off"     // Optional Signal
		record[16] = "1"       // DTMF ID
		record[17] = "1"       // 2Tone ID
		record[18] = "1"       // 5Tone ID
		record[19] = "Off"     // PTT ID
		record[20] = strconv.Itoa(c.ColorCode)
		record[21] = strconv.Itoa(c.TimeSlot)
		if c.TimeSlot == 0 {
			record[21] = "1"
		}
		record[22] = "None" // Scan List
		record[23] = c.RxGroup
		if record[23] == "" {
			record[23] = "None"
		}
		record[24] = "Off" // PTT Prohibit
		record[25] = "Off" // Reverse
		record[26] = "Off" // Digital Duplex
		record[27] = "Off" // Slot Suit
		record[28] = "Normal Encryption"
		record[29] = "Off"
		record[30] = "Off" // Call Conf
		if c.TalkAround {
			record[31] = "On"
		} else {
			record[31] = "Off" // Talk Around
		}
		if c.WorkAlone {
			record[32] = "On"
		} else {
			record[32] = "Off" // Work Alone
		}
		record[33] = "251.1" // Custom CTCSS default
		// ... Defaults for rest ...
		for j := 34; j < len(header); j++ {
			// Based on sample, most of these are "Off" or "0".
			// Working file showed "Idle TX" as "On" (index 36) for ANalog, but Off for Digital?
			// Digital sample has Off. Let's default Off.

			switch header[j] {
			case "Idle TX":
				record[j] = "Off"
			case "SMS Confirmation":
				record[j] = "Off"
			case "APRS Report Type", "Ranging", "APRS RX", "Analog APRS PTT Mode", "Digital APRS PTT Mode":
				record[j] = "Off"
			case "2TONE Decode":
				record[j] = "0"
			case "NxdnGroupId", "NxdnIdNum", "NxdnStateNum":
				record[j] = "0"
			case "Digital APRS Report Channel", "txcc":
				record[j] = "1"
			default:
				record[j] = "0"
			}
		}
		record[41] = "1" // Digital APRS Report Channel
		record[76] = "1" // txcc

		if err := writeAnyToneRecord(w, record); err != nil {
			return err
		}
	}
	return nil
}

func ExportAnyTone890Talkgroups(contacts []models.Contact, w io.Writer) error {
	// Manual write for quotes
	if err := writeAnyToneRecord(w, []string{"No.", "Radio ID", "Name", "Call Type", "Call Alert"}); err != nil {
		return err
	}

	// contacts slice passed in

	for i, c := range contacts {
		cType := "Group Call"
		if c.Type == models.ContactTypePrivate {
			cType = "Private Call"
		} else if c.Type == models.ContactTypeAllCall {
			cType = "All Call"
		}
		if err := writeAnyToneRecord(w, []string{strconv.Itoa(i + 1), strconv.Itoa(c.DMRID), c.Name, cType, "None"}); err != nil {
			return err
		}
	}
	return nil
}

func ExportAnyTone890Zones(zones []models.Zone, w io.Writer) error {
	if err := writeAnyToneRecord(w, []string{"No.", "Zone Name", "Zone Channel Member", "Zone Channel Member RX Frequency", "Zone Channel Member TX Frequency", "A Channel", "A Channel RX Frequency", "A Channel TX Frequency", "B Channel", "B Channel RX Frequency", "B Channel TX Frequency", "Zone Hide "}); err != nil {
		return err
	}

	// zones slice passed in

	for i, z := range zones {
		var chanNames []string
		for _, c := range z.Channels {
			chanNames = append(chanNames, c.Name)
		}
		// Simplified member string, actual AnyTone might need proper formatting
		memberStr := ""
		rxFreqStr := ""
		txFreqStr := ""

		if len(chanNames) > 0 {
			for j, name := range chanNames {
				if j > 0 {
					memberStr += "|"
					rxFreqStr += "|"
					txFreqStr += "|"
				}
				memberStr += name
				rxFreqStr += fmt.Sprintf("%.5f", z.Channels[j].RxFrequency)
				txFreqStr += fmt.Sprintf("%.5f", z.Channels[j].TxFrequency)
			}
		}

		aChan := ""
		aRx := ""
		aTx := ""
		bChan := ""
		bRx := ""
		bTx := ""

		if len(z.Channels) > 0 {
			aChan = z.Channels[0].Name
			aRx = fmt.Sprintf("%.5f", z.Channels[0].RxFrequency)
			aTx = fmt.Sprintf("%.5f", z.Channels[0].TxFrequency)
		}
		if len(z.Channels) > 1 {
			bChan = z.Channels[1].Name
			bRx = fmt.Sprintf("%.5f", z.Channels[1].RxFrequency)
			bTx = fmt.Sprintf("%.5f", z.Channels[1].TxFrequency)
		} else if len(z.Channels) > 0 {
			// If only 1 channel, duplicate to B? Or leave blank?
			// Sample shows B populated. Let's populate B with same if only 1 exists,
			// or typically Zone has >1. If only 1, AnyTone might require A and B.
			// Let's use Channel 0 for B as well if count is 1.
			bChan = z.Channels[0].Name
			bRx = fmt.Sprintf("%.5f", z.Channels[0].RxFrequency)
			bTx = fmt.Sprintf("%.5f", z.Channels[0].TxFrequency)
		}

		if err := writeAnyToneRecord(w, []string{
			strconv.Itoa(i + 1),
			z.Name,
			memberStr,
			rxFreqStr, txFreqStr, // RX/TX Freq lists
			aChan, aRx, aTx, // A Channel
			bChan, bRx, bTx, // B Channel
			"0", // Zone Hide
		}); err != nil {
			return err
		}
	}
	return nil
}

func ExportAnyTone890DigitalContacts(contacts []models.DigitalContact, w io.Writer) error {
	if err := writeAnyToneRecord(w, []string{"No.", "Radio ID", "Callsign", "Name", "City", "State", "Country", "Remarks", "Call Type", "Call Alert"}); err != nil {
		return err
	}

	// contacts slice passed in
	// Use manual loop
	for i, c := range contacts {
		// Offset index by 0 as we iterate slice
		// But note: batch logic in original version. Here we just stream.
		idx := i + 1
		if err := writeAnyToneRecord(w, []string{
			strconv.Itoa(idx),
			strconv.Itoa(c.DMRID),
			c.Callsign,
			c.Name,
			c.City,
			c.State,
			c.Country,
			c.Remarks,
			"Private Call",
			"None",
		}); err != nil {
			return err
		}
	}
	return nil
}

// Helper to write a record with forced quotes around every field
func writeAnyToneRecord(w io.Writer, record []string) error {
	for i, field := range record {
		if i > 0 {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		// Escape quotes: " -> ""
		escaped := strings.ReplaceAll(field, "\"", "\"\"")
		if _, err := fmt.Fprintf(w, "\"%s\"", escaped); err != nil {
			return err
		}
	}
	// Use CRLF for Windows/Radio compatibility
	if _, err := w.Write([]byte("\r\n")); err != nil {
		return err
	}
	return nil
}
