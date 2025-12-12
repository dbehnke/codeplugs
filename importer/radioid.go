package importer

import (
	"codeplugs/models"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
)

// ImportRadioIDCSV imports contacts from a RadioID.net user.csv
// processedIDs is an optional map of IDs to filter by (if nil, all are imported)
func ImportRadioIDCSV(r io.Reader, processedIDs map[int]bool) ([]models.DigitalContact, error) {
	reader := csv.NewReader(r)
	// Allow for variable number of fields, though typically standard
	reader.FieldsPerRecord = -1

	// Read and parse headers
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var contacts []models.DigitalContact

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Helper to try multiple keys
		getVal := func(keys ...string) string {
			for _, k := range keys {
				if idx, ok := headerMap[k]; ok && idx < len(record) {
					return strings.TrimSpace(record[idx])
				}
			}
			return ""
		}

		idStr := getVal("radio_id", "radio id", "id")
		if idStr == "" {
			continue
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue // Skip invalid IDs
		}

		// Filtering logic
		if processedIDs != nil {
			if !processedIDs[id] {
				continue // Skip if not in allowed list
			}
		}

		fname := getVal("first_name", "first name", "firstname")
		lname := getVal("last_name", "last name", "lastname")
		fullName := strings.TrimSpace(fname + " " + lname)
		callsign := getVal("callsign")
		city := getVal("city")
		state := getVal("state")
		country := getVal("country")
		remarks := getVal("remarks") // Sometimes "remarks" or "ipscl_qth" or similar, usually just "remarks" in radioid dump

		if fullName == "" {
			fullName = callsign // Fallback
		}

		contact := models.DigitalContact{
			Name:     fullName,
			Callsign: callsign,
			City:     city,
			State:    state,
			Country:  country,
			Remarks:  remarks,
			DMRID:    id,
		}
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

// ParseBrandmeisterLastHeard parses the active IDs from a BM CSV
func ParseBrandmeisterLastHeard(r io.Reader) (map[int]bool, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	idIdx := -1
	for i, h := range headers {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "sending id" || h == "radio id" || h == "id" {
			idIdx = i
			break
		}
	}

	// Fallback to 0 if not found? Similar to contactfilter logic
	if idIdx == -1 {
		idIdx = 0
	}

	ids := make(map[int]bool)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed lines or return error?
			// Best effort
			continue
		}
		if len(record) <= idIdx {
			continue
		}

		idStr := strings.TrimSpace(record[idIdx])
		if id, err := strconv.Atoi(idStr); err == nil {
			ids[id] = true
		}
	}
	return ids, nil
}
