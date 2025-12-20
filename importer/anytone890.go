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

func ImportAnyTone890Channels(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)

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

		c := models.Channel{}
		if idx, ok := headerMap["Channel Name"]; ok {
			c.Name = record[idx]
		}
		if idx, ok := headerMap["Receive Frequency"]; ok {
			c.RxFrequency, _ = strconv.ParseFloat(record[idx], 64)
		}
		if idx, ok := headerMap["Transmit Frequency"]; ok {
			c.TxFrequency, _ = strconv.ParseFloat(record[idx], 64)
		}
		if idx, ok := headerMap["Channel Type"]; ok {
			if record[idx] == "D-Digital" {
				c.Type = models.ChannelTypeDigitalDMR
				c.Protocol = models.ProtocolDMR
			} else {
				c.Type = models.ChannelTypeAnalog
				c.Protocol = models.ProtocolFM
			}
		}
		if idx, ok := headerMap["Transmit Power"]; ok {
			c.Power = record[idx]
		}
		if idx, ok := headerMap["Band Width"]; ok {
			c.Bandwidth = record[idx]
		}
		if idx, ok := headerMap["RX Color Code"]; ok {
			c.ColorCode, _ = strconv.Atoi(record[idx])
		}
		if idx, ok := headerMap["Slot"]; ok {
			c.TimeSlot, _ = strconv.Atoi(record[idx])
		}
		if idx, ok := headerMap["Receive Group List"]; ok {
			c.RxGroup = record[idx]
		}
		if idx, ok := headerMap["Contact/Talk Group"]; ok {
			c.TxContact = record[idx]
		}
		if idx, ok := headerMap["Scan List"]; ok {
			c.ScanList = record[idx]
		}
		if idx, ok := headerMap["Optional Signal"]; ok {
			c.OptionalSignal = record[idx]
		}
		if idx, ok := headerMap["DTMF ID"]; ok {
			c.DtmfID = record[idx]
		}
		if idx, ok := headerMap["2Tone ID"]; ok {
			c.Tone2ID = record[idx]
		}
		if idx, ok := headerMap["5Tone ID"]; ok {
			c.Tone5ID = record[idx]
		}
		if idx, ok := headerMap["PTT ID"]; ok {
			c.PttId = record[idx]
		}
		// AnyTone specific
		if idx, ok := headerMap["Talk Around(Simplex)"]; ok {
			c.TalkAround = parseAnyToneBool(record[idx])
		}
		if idx, ok := headerMap["Work Alone"]; ok {
			c.WorkAlone = parseAnyToneBool(record[idx])
		}

		channels = append(channels, c)
	}
	return db.Create(&channels).Error
}

func parseAnyToneBool(s string) bool {
	return strings.ToLower(s) == "on"
}

func ImportAnyTone890Talkgroups(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	_, err := reader.Read() // skip header
	if err != nil {
		return err
	}

	var contacts []models.Contact
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// "No.","Radio ID","Name","Call Type","Call Alert"
		if len(record) < 3 {
			continue
		}

		id, _ := strconv.Atoi(record[1])
		c := models.Contact{
			Name:  record[2],
			DMRID: id,
		}
		if strings.EqualFold(record[3], "Group Call") {
			c.Type = models.ContactTypeGroup
		} else if strings.EqualFold(record[3], "Private Call") {
			c.Type = models.ContactTypePrivate
		} else {
			c.Type = models.ContactTypeAllCall
		}
		contacts = append(contacts, c)
	}

	for _, c := range contacts {
		if err := db.Where("dmr_id = ? AND type = ?", c.DMRID, c.Type).FirstOrCreate(&c).Error; err != nil {
			fmt.Printf("Error importing contact %s: %v\n", c.Name, err)
		}
	}
	return nil
}

func ImportAnyTone890Zones(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.TrimSpace(h)] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// "No.","Zone Name","Zone Channel Member",...
		zName := ""
		if idx, ok := headerMap["Zone Name"]; ok {
			zName = record[idx]
		}
		if zName == "" {
			continue
		}

		zone, err := models.FindOrCreateZone(db, zName)
		if err != nil {
			return err
		}

		if idx, ok := headerMap["Zone Channel Member"]; ok {
			rawMembers := record[idx]
			if rawMembers != "" {
				members := strings.Split(rawMembers, "|")
				var channels []models.Channel
				db.Where("name IN ?", members).Find(&channels)
				db.Model(zone).Association("Channels").Append(&channels)
			}
		}
	}
	return nil
}

func ImportAnyTone890DigitalContacts(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	_, err := reader.Read()
	if err != nil {
		return err
	}

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

		// "No.","Radio ID","Callsign","Name","City","State","Country","Remarks","Call Type","Call Alert"
		if len(record) < 8 {
			continue
		}

		id, _ := strconv.Atoi(record[1])
		dc := models.DigitalContact{
			DMRID:    id,
			Callsign: record[2],
			Name:     record[3],
			City:     record[4],
			State:    record[5],
			Country:  record[6],
			Remarks:  record[7],
		}
		contacts = append(contacts, dc)

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

func ImportAnyTone890ScanLists(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.TrimSpace(h)] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// "No.","Scan List Name","Scan Channel Member",...
		if idx, ok := headerMap["Scan List Name"]; ok {
			name := record[idx]
			list, err := models.FindOrCreateScanList(db, name)
			if err != nil {
				return err
			}

			if idxM, ok := headerMap["Scan Channel Member"]; ok {
				members := strings.Split(record[idxM], "|")
				var channels []models.Channel
				db.Where("name IN ?", members).Find(&channels)
				db.Model(list).Association("Channels").Append(&channels)
			}
		}
	}
	return nil
}

func ImportAnyTone890RoamingChannels(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.TrimSpace(h)] = i
	}

	var channels []models.RoamingChannel
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		rc := models.RoamingChannel{}
		if idx, ok := headerMap["Name"]; ok {
			rc.Name = record[idx]
		}
		if idx, ok := headerMap["RX Frequency"]; ok {
			rc.RxFrequency, _ = strconv.ParseFloat(record[idx], 64)
		} else if idx, ok := headerMap["Receive Frequency"]; ok {
			rc.RxFrequency, _ = strconv.ParseFloat(record[idx], 64)
		}

		if idx, ok := headerMap["TX Frequency"]; ok {
			rc.TxFrequency, _ = strconv.ParseFloat(record[idx], 64)
		} else if idx, ok := headerMap["Transmit Frequency"]; ok {
			rc.TxFrequency, _ = strconv.ParseFloat(record[idx], 64)
		}
		if idx, ok := headerMap["Color Code"]; ok {
			rc.ColorCode, _ = strconv.Atoi(record[idx])
		}
		if idx, ok := headerMap["Slot"]; ok {
			rc.TimeSlot, _ = strconv.Atoi(record[idx])
		}

		if rc.Name != "" {
			channels = append(channels, rc)
		}
	}

	for _, ch := range channels {
		// Upsert based on Name and Freq
		var existing models.RoamingChannel
		if err := db.Where("name = ? AND rx_frequency = ?", ch.Name, ch.RxFrequency).First(&existing).Error; err == nil {
			ch.ID = existing.ID
			db.Save(&ch)
		} else {
			db.Create(&ch)
		}
	}
	return nil
}

func ImportAnyTone890RoamingZones(db *gorm.DB, r io.Reader) error {
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.TrimSpace(h)] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// "No.","Name","Roaming Channel Member"
		if idx, ok := headerMap["Name"]; ok {
			name := record[idx]
			if name == "" {
				continue
			}
			
			zone, err := models.FindOrCreateRoamingZone(db, name)
			if err != nil {
				return err
			}

			if idxM, ok := headerMap["Roaming Channel Member"]; ok {
				members := strings.Split(record[idxM], "|")
				var channels []models.RoamingChannel
				db.Where("name IN ?", members).Find(&channels)
				db.Model(zone).Association("Channels").Append(&channels)
			}
		}
	}
	return nil
}
