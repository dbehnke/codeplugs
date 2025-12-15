package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"codeplugs/models"
)

// ImportChirpCSV imports channels from a Chirp-formatted CSV stream.
func ImportChirpCSV(r io.Reader) ([]models.Channel, error) {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true
	headers, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[h] = i
	}

	var channels []models.Channel

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		getVal := func(colName string) string {
			if idx, ok := headerMap[colName]; ok && idx < len(record) {
				return record[idx]
			}
			return ""
		}

		channel := models.Channel{}

		// Chirp standard fields
		// Location,Name,Frequency,Duplex,Offset,Tone,rToneFreq,cToneFreq,DtcsCode,DtcsPolarity,Mode,TStep,Skip,Comment,URCALL,RPT1CALL,RPT2CALL,DVCODE

		channel.Name = getVal("Name")

		freqStr := getVal("Frequency")
		if freqStr != "" {
			channel.RxFrequency, _ = strconv.ParseFloat(freqStr, 64)
		}

		// Calculate Tx Freq based on Duplex and Offset
		duplex := getVal("Duplex")
		offsetStr := getVal("Offset")
		offset, _ := strconv.ParseFloat(offsetStr, 64)

		fmt.Printf("Debug: Name=%s Duplex=%s Offset=%s OffsetVal=%f\n", channel.Name, duplex, offsetStr, offset)

		switch duplex {
		case "+":
			channel.TxFrequency = channel.RxFrequency + offset
		case "-":
			channel.TxFrequency = channel.RxFrequency - offset
		default:
			channel.TxFrequency = channel.RxFrequency // Simplex or "off"
		}

		channel.Mode = getVal("Mode")
		if channel.Mode == "NFM" {
			channel.Mode = "FM"
		}

		// Tone mapping
		toneMode := getVal("Tone") // "Tone", "TSQL", "DTCS", "Cross"
		switch toneMode {
		case "Tone":
			channel.Tone = getVal("rToneFreq")
		case "TSQL":
			channel.Tone = getVal("cToneFreq")
		case "DTCS":
			channel.Tone = getVal("DtcsCode")
		}

		channel.Notes = getVal("Comment")

		channels = append(channels, channel)
	}

	return channels, nil
}
