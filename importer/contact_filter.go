package importer

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"codeplugs/models"

	"gorm.io/gorm"
)

// LoadFilterList loads a list of allowed DMR IDs from a file.
// Supports CSV with a header containing "ID" or similar, or a plain list of IDs.
// Returns a map for O(1) lookup.
func LoadFilterList(path string) (map[int]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Heuristic: Peek first few lines to detect format?
	// Or just try CSV parse.

	allowed := make(map[int]bool)

	// Try reading as CSV
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // flexible
	r.LazyQuotes = true

	records, err := r.ReadAll()
	if err != nil {
		// Fallback: Read line by line as plain text if CSV fails significantly
		// Reset file
		f.Seek(0, 0)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			// Extract first number found?
			fields := strings.FieldsFunc(line, func(r rune) bool {
				return r == ',' || r == ';' || r == ' ' || r == '\t'
			})
			for _, field := range fields {
				if id, err := strconv.Atoi(strings.TrimSpace(field)); err == nil && id > 0 {
					allowed[id] = true
					break // Assume one ID per line?
				}
			}
		}
		return allowed, nil
	}

	if len(records) == 0 {
		return allowed, nil
	}

	header := records[0]
	idCol := -1

	// Find ID column
	for i, col := range header {
		lcol := strings.ToLower(col)
		if strings.Contains(lcol, "radio id") || strings.Contains(lcol, "dmr id") || lcol == "id" {
			idCol = i
			break
		}
	}

	startRow := 1
	if idCol == -1 {
		// No header found? Check if first row is ID
		if res, err := strconv.Atoi(records[0][0]); err == nil && res > 0 {
			// Probably no header, just data?
			idCol = 0
			startRow = 0
		} else {
			// Maybe it has a header but we missed it?
			// Fallback to checking line by line?
			return nil, fmt.Errorf("could not detect ID column in header: %v", header)
		}
	}

	for i := startRow; i < len(records); i++ {
		row := records[i]
		if len(row) <= idCol {
			continue
		}

		val := row[idCol]
		if id, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && id > 0 {
			allowed[id] = true
		}
	}

	return allowed, nil
}

// ImportFilterListToDB loads a filter list from a file and saves it to the database.
// It overwrites any existing list with the same name.
func ImportFilterListToDB(db *gorm.DB, path string, listName string) error {
	// reuse logic to get map of IDs
	allowed, err := LoadFilterList(path)
	if err != nil {
		return err
	}

	if len(allowed) == 0 {
		return fmt.Errorf("no IDs found in file %s", path)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 1. Find or Create List
		var list models.ContactList
		if err := tx.Where("name = ?", listName).First(&list).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				list = models.ContactList{Name: listName, Description: fmt.Sprintf("Imported from %s", path)}
				if err := tx.Create(&list).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// 2. Clear existing entries (Overwrite mode)
		if err := tx.Where("contact_list_id = ?", list.ID).Delete(&models.ContactListEntry{}).Error; err != nil {
			return err
		}

		// 3. Batch Insert New Entries
		batchSize := 1000
		entries := make([]models.ContactListEntry, 0, batchSize)
		count := 0
		for id := range allowed {
			entries = append(entries, models.ContactListEntry{
				ContactListID: list.ID,
				DMRID:         id,
			})
			count++

			if len(entries) >= batchSize {
				if err := tx.Create(&entries).Error; err != nil {
					return err
				}
				entries = entries[:0]
			}
		}
		if len(entries) > 0 {
			if err := tx.Create(&entries).Error; err != nil {
				return err
			}
		}

		fmt.Printf("Imported %d entries into list '%s'.\n", count, listName)
		return nil
	})
}
