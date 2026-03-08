package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"codeplugs/importer"
)

// GenerateContacts generates a filtered contact list from a source CSV.
// It loads DMR IDs from the filter file and only includes contacts from the
// source file that match those IDs.
//
// Parameters:
//   - filterFile: Path to CSV file containing DMR IDs to filter by
//   - sourceFile: Path to source CSV (RadioID.net format)
//   - outputFile: Path to output filtered CSV
//   - format: Output format - "radioid" (default) or "dm32uv"
//
// Source CSV format (RadioID.net):
//
//	radio_id,callsign,first_name,last_name,city,state,country,remarks
//
// Output formats:
//   - radioid: Same as source format
//   - dm32uv: No.,ID,Repeater,Name,City,Province,Country,Remark,Type,Alert Call
//   - at890: "No.","Radio ID","Callsign","Name","City","State","Country","Remarks","Call Type","Call Alert" (quoted)
//
// Filter CSV format:
//
//	Can be a simple list of IDs or CSV with "Radio ID", "DMR ID", or "id" column
func GenerateContacts(filterFile, sourceFile, outputFile, format string) error {
	// Load filter list
	allowedIDs, err := importer.LoadFilterList(filterFile)
	if err != nil {
		return fmt.Errorf("failed to load filter list: %w", err)
	}

	if len(allowedIDs) == 0 {
		return fmt.Errorf("no IDs found in filter file: %s", filterFile)
	}

	fmt.Printf("Loaded %d IDs from filter file\n", len(allowedIDs))

	// Open source file
	srcFile, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Read source CSV
	reader := csv.NewReader(srcFile)
	reader.FieldsPerRecord = -1 // Allow variable fields

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Find column indices in source
	idCol := -1
	callsignCol := -1
	firstNameCol := -1
	lastNameCol := -1
	cityCol := -1
	stateCol := -1
	countryCol := -1
	remarksCol := -1

	for i, col := range header {
		lowerCol := strings.ToLower(col)
		switch lowerCol {
		case "radio_id", "radio id", "dmr_id", "dmr id", "id":
			idCol = i
		case "callsign":
			callsignCol = i
		case "first_name", "first name":
			firstNameCol = i
		case "last_name", "last name":
			lastNameCol = i
		case "city":
			cityCol = i
		case "state":
			stateCol = i
		case "country":
			countryCol = i
		case "remarks", "remark":
			remarksCol = i
		}
	}

	if idCol == -1 {
		return fmt.Errorf("could not find ID column in header: %v", header)
	}

	// Create writer based on format
	var writer *csv.Writer
	var writeRecord func([]string) error

	switch format {
	case "dm32uv":
		writer = csv.NewWriter(outFile)
		writer.Write([]string{"No.", "ID", "Repeater", "Name", "City", "Province", "Country", "Remark", "Type", "Alert Call"})
		writeRecord = func(record []string) error {
			return writer.Write(record)
		}
	case "at890":
		// AnyTone 890 uses forced quotes and CRLF
		writeRecord = func(record []string) error {
			for i, field := range record {
				if i > 0 {
					if _, err := outFile.Write([]byte(",")); err != nil {
						return err
					}
				}
				escaped := strings.ReplaceAll(field, "\"", "\"\"")
				if _, err := fmt.Fprintf(outFile, "\"%s\"", escaped); err != nil {
					return err
				}
			}
			if _, err := outFile.Write([]byte("\r\n")); err != nil {
				return err
			}
			return nil
		}
		// Write header
		writeRecord([]string{"No.", "Radio ID", "Callsign", "Name", "City", "State", "Country", "Remarks", "Call Type", "Call Alert"})
	default: // radioid
		writer = csv.NewWriter(outFile)
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
		writeRecord = func(record []string) error {
			return writer.Write(record)
		}
	}

	// Process records
	processed := 0
	included := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		processed++

		if len(record) <= idCol {
			continue
		}

		idStr := strings.TrimSpace(record[idCol])
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}

		if allowedIDs[id] {
			idx := included + 1
			callsign := getColOrDefault(record, callsignCol, "")
			firstName := getColOrDefault(record, firstNameCol, "")
			lastName := getColOrDefault(record, lastNameCol, "")
			name := strings.TrimSpace(firstName + " " + lastName)
			if name == "" {
				name = callsign
			}

			var outputRecord []string
			switch format {
			case "dm32uv":
				outputRecord = []string{
					strconv.Itoa(idx),
					idStr,
					callsign,
					name,
					getColOrDefault(record, cityCol, ""),
					getColOrDefault(record, stateCol, ""),
					getColOrDefault(record, countryCol, ""),
					getColOrDefault(record, remarksCol, ""),
					"Private Call",
					"0",
				}
			case "at890":
				outputRecord = []string{
					strconv.Itoa(idx),
					idStr,
					callsign,
					name,
					getColOrDefault(record, cityCol, ""),
					getColOrDefault(record, stateCol, ""),
					getColOrDefault(record, countryCol, ""),
					getColOrDefault(record, remarksCol, ""),
					"Private Call",
					"None",
				}
			default:
				outputRecord = record
			}

			if err := writeRecord(outputRecord); err != nil {
				return fmt.Errorf("failed to write record: %w", err)
			}
			included++
		}
	}

	if writer != nil {
		writer.Flush()
		if err := writer.Error(); err != nil {
			return fmt.Errorf("error flushing writer: %w", err)
		}
	}

	fmt.Printf("Processed %d records, included %d in output\n", processed, included)
	fmt.Printf("Filtered contacts written to: %s (format: %s)\n", outputFile, format)

	return nil
}

func getColOrDefault(record []string, col int, defaultVal string) string {
	if col >= 0 && col < len(record) {
		return record[col]
	}
	return defaultVal
}
