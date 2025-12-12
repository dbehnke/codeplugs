package exporter

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"codeplugs/models"

	"gorm.io/gorm"
)

func ExportDM32UV(db *gorm.DB, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	f1, err := os.Create(filepath.Join(outputDir, "channels.csv"))
	if err != nil {
		return err
	}
	defer f1.Close()
	var channels []models.Channel
	if err := db.Find(&channels).Error; err != nil {
		return err
	}
	if err := ExportDM32UVChannels(channels, f1); err != nil {
		return err
	}

	f2, err := os.Create(filepath.Join(outputDir, "talkgroups.csv"))
	if err != nil {
		return err
	}
	defer f2.Close()
	var talkgroups []models.Contact
	if err := db.Where("type IN ?", []models.ContactType{models.ContactTypeGroup, models.ContactTypePrivate, models.ContactTypeAllCall}).Find(&talkgroups).Error; err != nil {
		return err
	}
	if err := ExportDM32UVTalkgroups(talkgroups, f2); err != nil {
		return err
	}

	f3, err := os.Create(filepath.Join(outputDir, "zones.csv"))
	if err != nil {
		return err
	}
	defer f3.Close()
	var zones []models.Zone
	if err := db.Preload("Channels").Find(&zones).Error; err != nil {
		return err
	}
	if err := ExportDM32UVZones(zones, f3); err != nil {
		return err
	}

	f4, err := os.Create(filepath.Join(outputDir, "digital_contacts.csv"))
	if err != nil {
		return err
	}
	defer f4.Close()
	var digitalContacts []models.DigitalContact
	// Fetch all for bulk export - might be large
	if err := db.Find(&digitalContacts).Error; err != nil {
		return err
	}
	if err := ExportDM32UVDigitalContacts(digitalContacts, f4); err != nil {
		return err
	}

	return nil
}

func ExportDM32UVChannels(channels []models.Channel, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header from sample
	header := []string{
		"No.", "Channel Name", "Channel Type", "RX Frequency[MHz]", "TX Frequency[MHz]", "Power", "Band Width", "Scan List", "TX Admit", "Emergency System", "Squelch Level", "APRS Report Type", "Forbid TX", "APRS Receive", "Forbid Talkaround", "Auto Scan", "Lone Work", "Emergency Indicator", "Emergency ACK", "Analog APRS PTT Mode", "Digital APRS PTT Mode", "TX Contact", "RX Group List", "Color Code", "Time Slot", "Encryption", "Encryption ID", "APRS Report Channel", "Direct Dual Mode", "Private Confirm", "Short Data Confirm", "DMR ID", "CTC/DCS Decode", "CTC/DCS Encode", "Scramble", "RX Squelch Mode", "Signaling Type", "PTT ID", "VOX Function", "PTT ID Display",
	}

	writer.Write(header)

	for i, c := range channels {
		record := make([]string, len(header))
		record[0] = strconv.Itoa(i + 1)
		record[1] = c.Name
		if c.Type == models.ChannelTypeDigital {
			record[2] = "Digital"
		} else {
			record[2] = "Analog"
		}
		record[3] = fmt.Sprintf("%.5f", c.RxFrequency)
		record[4] = fmt.Sprintf("%.5f", c.TxFrequency)
		record[5] = c.Power
		if c.Power == "" {
			record[5] = "High"
		} // Default
		record[6] = c.Bandwidth
		if c.Bandwidth == "" {
			record[6] = "12.5KHz"
		}
		record[7] = "None"     // Scan List support todo
		record[8] = "Allow TX" // Default
		if c.ForbidTx {
			record[8] = "Forbid TX"
		} // Logic check needed
		record[9] = "None"
		record[10] = strconv.Itoa(c.SquelchLevel)
		if c.SquelchLevel == 0 {
			record[10] = "3"
		} // Default
		record[11] = c.AprsReportType
		if c.AprsReportType == "" {
			record[11] = "Off"
		}
		record[12] = boolToIntStr(c.ForbidTx)
		record[13] = boolToIntStr(c.AprsReceive)
		record[14] = boolToIntStr(c.ForbidTalkaround)
		record[15] = boolToIntStr(c.AutoScan)
		record[16] = boolToIntStr(c.LoneWork)
		record[17] = boolToIntStr(c.EmergencyIndicator)
		record[18] = boolToIntStr(c.EmergencyAck)
		record[19] = strconv.Itoa(c.AnalogAprsPttMode)
		record[20] = strconv.Itoa(c.DigitalAprsPttMode)
		record[21] = c.TxContact
		if c.TxContact == "" {
			record[21] = "None"
		}
		record[22] = c.RxGroup
		if c.RxGroup == "" {
			record[22] = "None"
		}
		record[23] = strconv.Itoa(c.ColorCode)
		record[24] = fmt.Sprintf("Slot %d", c.TimeSlot)
		if c.TimeSlot == 0 {
			record[24] = "Slot 1"
		}
		record[25] = "0" // Encryption default
		record[26] = "None"
		record[27] = "1" // APRS Report Channel default
		record[28] = boolToIntStr(c.DirectDualMode)
		record[29] = boolToIntStr(c.PrivateConfirm)
		record[30] = boolToIntStr(c.ShortDataConfirm)
		record[31] = "None" // DMR ID todo - should be from global settings or per channel? Sample has "KF8S Dave"
		record[32] = c.RxTone
		if c.RxTone == "" {
			record[32] = "None"
		}
		record[33] = c.TxTone
		if c.TxTone == "" {
			record[33] = "None"
		}
		record[34] = "None"
		record[35] = "Carrier/CTC" // Default
		record[36] = "None"
		record[37] = "OFF"
		record[38] = "0"
		record[39] = "0"

		writer.Write(record)
	}
	return nil
}

func boolToIntStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func ExportDM32UVTalkgroups(contacts []models.Contact, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"No.", "Name", "ID", "Type"})

	// contacts slice passed in

	for i, c := range contacts {
		cType := "Group Call"
		if c.Type == models.ContactTypePrivate {
			cType = "Private Call"
		} else if c.Type == models.ContactTypeAllCall {
			cType = "All Call"
		}
		writer.Write([]string{strconv.Itoa(i + 1), c.Name, strconv.Itoa(c.DMRID), cType})
	}
	return nil
}

func ExportDM32UVZones(zones []models.Zone, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"No.", "Zone Name", "Channel Members"})

	// zones slice passed in

	for i, z := range zones {
		var chanNames []string
		for _, c := range z.Channels {
			chanNames = append(chanNames, c.Name)
		}
		writer.Write([]string{strconv.Itoa(i + 1), z.Name, strings.Join(chanNames, "|")})
	}
	return nil
}

func ExportDM32UVDigitalContacts(contacts []models.DigitalContact, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"No.", "ID", "Repeater", "Name", "City", "Province", "Country", "Remark", "Type", "Alert Call"})

	// contacts slice passed in
	// Use manual loop instead of FindInBatches since we have slice
	for i, c := range contacts {
		// Index 1-based
		idx := i + 1
		writer.Write([]string{
			strconv.Itoa(idx),
			strconv.Itoa(c.DMRID),
			c.Callsign,
			c.Name,
			c.City,
			c.State,
			c.Country,
			c.Remarks,
			"Private Call",
			"0",
		})
	}
	return nil
}
