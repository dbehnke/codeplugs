package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"codeplugs/models"

	"gorm.io/gorm"
)

func ImportDM32UVChannels(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	// Typically DM32UV CSVs might use specific settings, but encoding/csv defaults are usually fine for standard CSVs.
	// If comma is different, adjust here.

	// Read Header
	header, err := reader.Read()
	if err != nil {
		return err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.TrimSpace(h)] = i
	}

	var channels []models.Channel

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		channel := models.Channel{}

		// Map fields based on headerMap
		if idx, ok := headerMap["Channel Name"]; ok {
			channel.Name = record[idx]
		}
		if idx, ok := headerMap["RX Frequency[MHz]"]; ok {
			f, _ := strconv.ParseFloat(record[idx], 64)
			channel.RxFrequency = f
		}
		if idx, ok := headerMap["TX Frequency[MHz]"]; ok {
			f, _ := strconv.ParseFloat(record[idx], 64)
			channel.TxFrequency = f
		}
		if idx, ok := headerMap["Channel Type"]; ok {
			cType := record[idx]
			if cType == "Digital" {
				channel.Type = models.ChannelTypeDigital
				channel.Protocol = models.ProtocolDMR
			} else {
				channel.Type = models.ChannelTypeAnalog
				channel.Protocol = models.ProtocolFM
			}
		}
		if idx, ok := headerMap["Power"]; ok {
			channel.Power = record[idx]
		}
		if idx, ok := headerMap["Band Width"]; ok {
			channel.Bandwidth = record[idx]
		}
		if idx, ok := headerMap["Color Code"]; ok {
			cc, _ := strconv.Atoi(record[idx])
			channel.ColorCode = cc
		}
		if idx, ok := headerMap["Time Slot"]; ok {
			tsStr := record[idx]
			if strings.Contains(tsStr, "Slot 1") {
				channel.TimeSlot = 1
			} else if strings.Contains(tsStr, "Slot 2") {
				channel.TimeSlot = 2
			}
		}
		if idx, ok := headerMap["RX Group List"]; ok {
			channel.RxGroup = record[idx]
		}
		if idx, ok := headerMap["TX Contact"]; ok {
			channel.TxContact = record[idx]
		}
		// Squelch Level
		if idx, ok := headerMap["Squelch Level"]; ok {
			sl, _ := strconv.Atoi(record[idx])
			channel.SquelchLevel = sl
		}
		// ... Map other fields ...
		if idx, ok := headerMap["CTC/DCS Decode"]; ok {
			channel.RxTone = record[idx]
			channel.RxDCS = record[idx] // Logic to differentiate might be needed
			channel.CtcDcsDecode = record[idx]
		}
		if idx, ok := headerMap["CTC/DCS Encode"]; ok {
			channel.TxTone = record[idx]
			channel.TxDCS = record[idx]
			channel.CtcDcsEncode = record[idx]
		}

		// Additional DM32UV fields
		if idx, ok := headerMap["APRS Report Type"]; ok {
			channel.AprsReportType = record[idx]
		}
		if idx, ok := headerMap["Forbid TX"]; ok {
			channel.ForbidTx = parseBool(record[idx])
		}
		if idx, ok := headerMap["APRS Receive"]; ok {
			channel.AprsReceive = parseBool(record[idx])
		}
		if idx, ok := headerMap["Forbid Talkaround"]; ok {
			channel.ForbidTalkaround = parseBool(record[idx])
		}
		if idx, ok := headerMap["Auto Scan"]; ok {
			channel.AutoScan = parseBool(record[idx])
		}
		if idx, ok := headerMap["Lone Work"]; ok {
			channel.LoneWork = parseBool(record[idx])
		}
		if idx, ok := headerMap["Emergency Indicator"]; ok {
			channel.EmergencyIndicator = parseBool(record[idx])
		}
		if idx, ok := headerMap["Emergency ACK"]; ok {
			channel.EmergencyAck = parseBool(record[idx])
		}
		// ... more fields mapping

		channels = append(channels, channel)
	}

	// Batch insert
	return db.Create(&channels).Error
}

func parseBool(s string) bool {
	// "0" or "Off" -> false, "1" or "On" -> true
	s = strings.ToLower(s)
	return s == "1" || s == "on" || s == "true" || s == "allow tx" // "Allow TX" logic might be inverted for "Forbid TX"
}

func ImportDM32UVTalkgroups(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	header, err := reader.Read() // skip header
	if err != nil {
		return err
	}
	_ = header

	var contacts []models.Contact
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// Expecting: No.,Name,ID,Type
		// e.g., 1,DMR Tech Net,5031268,Group Call
		if len(record) < 4 {
			continue
		}

		id, _ := strconv.Atoi(record[2])
		contact := models.Contact{
			Name:  record[1],
			DMRID: id,
		}

		if strings.EqualFold(record[3], "Group Call") {
			contact.Type = models.ContactTypeGroup
		} else if strings.EqualFold(record[3], "Private Call") {
			contact.Type = models.ContactTypePrivate
		} else {
			contact.Type = models.ContactTypeAllCall
		}

		contacts = append(contacts, contact)
	}

	// Upsert contacts
	for _, c := range contacts {
		if err := db.Where("dmr_id = ? AND type = ?", c.DMRID, c.Type).FirstOrCreate(&c).Error; err != nil {
			fmt.Printf("Error importing contact %s: %v\n", c.Name, err)
		}
	}
	return nil
}

func ImportDM32UVZones(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	_, err := reader.Read() // skip header
	if err != nil {
		return err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// No.,Zone Name,Channel Members
		// 1,Home,KF8S|KF8S TS1|...
		if len(record) < 3 {
			continue
		}

		zoneName := record[1]
		zone, err := models.FindOrCreateZone(db, zoneName)
		if err != nil {
			return err
		}

		channelNames := strings.Split(record[2], "|")
		var channels []models.Channel
		db.Where("name IN ?", channelNames).Find(&channels)

		err = db.Model(zone).Association("Channels").Append(&channels)
		if err != nil {
			return err
		}
	}
	return nil
}

func ImportDM32UVDigitalContacts(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	_, err := reader.Read() // skip header
	if err != nil {
		return err
	}

	// Batch insert for performance
	var contacts []models.DigitalContact
	batchSize := 1000

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// No.,ID,Repeater,Name,City,Province,Country,Remark,Type,Alert Call
		// 1,2020005,SV3SMG,Ioannis,Kalamata,Peloponnisos,Greece,,Private Call,0
		if len(record) < 8 {
			continue
		}

		id, _ := strconv.Atoi(record[1])
		contact := models.DigitalContact{
			DMRID:    id,
			Callsign: record[2], // Repeater/Callsign column
			Name:     record[3],
			City:     record[4],
			State:    record[5],
			Country:  record[6],
			Remarks:  record[7],
		}
		contacts = append(contacts, contact)

		if len(contacts) >= batchSize {
			if err := db.Save(&contacts).Error; err != nil {
				return err
			}
			contacts = nil
		}
	}

	if len(contacts) > 0 {
		return db.Save(&contacts).Error
	}
	return nil
}
