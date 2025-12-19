package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"codeplugs/database"
	"codeplugs/exporter"
	"codeplugs/importer"
	"codeplugs/models"
	"codeplugs/services"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func HandleChannels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		var channels []models.Channel
		database.DB.Order("sort_order asc").Find(&channels)
		json.NewEncoder(w).Encode(channels)
	case "POST":
		var ch models.Channel
		if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if ch.ID == 0 {
			database.DB.Create(&ch)
		} else {
			database.DB.Save(&ch)
		}
		json.NewEncoder(w).Encode(ch)
	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			database.DB.Delete(&models.Channel{}, id)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func HandleChannelReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Reordering %d channels via SortOrder...", len(req.IDs))

	tx := database.DB.Begin()
	if tx.Error != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update SortOrder for each ID
	for i, id := range req.IDs {
		// i is 0-based index, we can use it directly as order (or i+1)
		// Batch update is efficient, but simple iteration is fine for SQLite.
		if err := tx.Model(&models.Channel{}).Where("id = ?", id).Update("sort_order", i+1).Error; err != nil {
			tx.Rollback()
			log.Printf("Error updating SortOrder for channel %d: %v", id, err)
			http.Error(w, "Failed to update channel order", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Transaction commit failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(100 << 20)

	format := r.FormValue("format")
	sourceMode := r.FormValue("source_mode")

	var path string
	var tempFile *os.File
	var err error

	if sourceMode != "download" {
		if format == "zip" {
			file, _, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Error retrieving file", http.StatusBadRequest)
				return
			}
			defer file.Close()

			buf := new(bytes.Buffer)
			io.Copy(buf, file)
			bytesReader := bytes.NewReader(buf.Bytes())

			zipReader, err := zip.NewReader(bytesReader, int64(buf.Len()))
			if err != nil {
				http.Error(w, "Invalid zip file", http.StatusBadRequest)
				return
			}

			filesMap := make(map[string]*zip.File)
			for _, f := range zipReader.File {
				filesMap[f.Name] = f
			}

			if f, ok := filesMap["digital_contacts.csv"]; ok {
				rc, _ := f.Open()
				err := importer.ImportDM32UVDigitalContacts(database.DB, rc)
				rc.Close()
				if err != nil {
					log.Printf("Error importing digital contacts: %v", err)
					http.Error(w, fmt.Sprintf("Error importing digital contacts: %v", err), http.StatusBadRequest)
					return
				}
			}

			if f, ok := filesMap["talkgroups.csv"]; ok {
				rc, _ := f.Open()
				err := importer.ImportDM32UVTalkgroups(database.DB, rc)
				rc.Close()
				if err != nil {
					log.Printf("Error importing talkgroups: %v", err)
					http.Error(w, fmt.Sprintf("Error importing talkgroups: %v", err), http.StatusBadRequest)
					return
				}
			}

			if f, ok := filesMap["channels.csv"]; ok {
				rc, _ := f.Open()
				err := importer.ImportDM32UVChannels(database.DB, rc)
				rc.Close()
				if err != nil {
					log.Printf("Error importing channels: %v", err)
					http.Error(w, fmt.Sprintf("Error importing channels: %v", err), http.StatusBadRequest)
					return
				}
			}

			if f, ok := filesMap["zones.csv"]; ok {
				rc, _ := f.Open()
				err := importer.ImportDM32UVZones(database.DB, rc)
				rc.Close()
				if err != nil {
					log.Printf("Error importing zones: %v", err)
					http.Error(w, fmt.Sprintf("Error importing zones: %v", err), http.StatusBadRequest)
					return
				}
			}

			fmt.Fprintf(w, "Zip Import Complete")
			return
		}

		if format == "db" {
			file, _, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Error retrieving file", http.StatusBadRequest)
				return
			}
			defer file.Close()

			tempFile, err := os.CreateTemp("", "restore-*.db")
			if err != nil {
				http.Error(w, "Error creating temp file", http.StatusInternalServerError)
				return
			}
			tempName := tempFile.Name()
			defer os.Remove(tempName)

			if _, err := io.Copy(tempFile, file); err != nil {
				tempFile.Close()
				http.Error(w, "Error saving file", http.StatusInternalServerError)
				return
			}
			tempFile.Close()

			database.Close()
			targetDB := "codeplugs.db" // TODO: Use actual configured path
			os.Rename(targetDB, targetDB+".bak")

			src, _ := os.Open(tempName)
			dst, _ := os.Create(targetDB)
			io.Copy(dst, src)
			src.Close()
			dst.Close()

			database.Connect(targetDB)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Database restored successfully. Please refresh.",
			})
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Error retrieving file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		tempFile, err = os.CreateTemp("", "upload-*.csv")
		if err != nil {
			http.Error(w, "Error creating temp file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		if _, err := io.Copy(tempFile, file); err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}
		path = tempFile.Name()
	}

	if format == "single" {
		importType := r.FormValue("import_type")
		radioPlatform := r.FormValue("radio_platform")
		overwrite := r.FormValue("overwrite") == "true"

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Error retrieving file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		tempFile, err := os.CreateTemp("", "import-single-*.csv")
		if err != nil {
			http.Error(w, "Error creating temp file", http.StatusInternalServerError)
			return
		}
		path := tempFile.Name()
		defer os.Remove(path)
		defer tempFile.Close()

		if _, err := io.Copy(tempFile, file); err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}

		f, err := os.Open(path)
		if err != nil {
			http.Error(w, "Error opening temp file", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		CurrentProgress.mu.Lock()
		CurrentProgress.Total = 0
		CurrentProgress.Processed = 0
		CurrentProgress.Status = "running"
		CurrentProgress.Message = fmt.Sprintf("Importing %s...", importType)
		CurrentProgress.mu.Unlock()
		BroadcastProgress()

		var count int
		var skipped int

		switch importType {
		case "channels":
			if overwrite {
				database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Channel{})
				database.DB.Exec("DELETE FROM sqlite_sequence WHERE name = 'channels'")
			}

			var channels []models.Channel
			var err error

			switch radioPlatform {
			case "dm32uv":
				err = importer.ImportDM32UVChannels(database.DB, f)
			case "at890":
				err = importer.ImportAnyTone890Channels(database.DB, f)
			default:
				channels, err = importer.ImportChannelsCSV(f)
				if err != nil || len(channels) == 0 {
					f.Seek(0, 0)
					chirpChannels, chirpErr := importer.ImportChirpCSV(f)
					if chirpErr == nil && len(chirpChannels) > 0 {
						channels = chirpChannels
						err = nil
					}
				}
				if err == nil {
					services.ResolveContacts(database.DB, channels)
					for _, ch := range channels {
						if !overwrite {
							var existing models.Channel
							if database.DB.Where("name = ? AND rx_frequency = ?", ch.Name, ch.RxFrequency).First(&existing).Error == nil {
								skipped++
								continue
							}
						}
						if res := database.DB.Create(&ch); res.Error == nil {
							count++
						}
					}
				}
			}

			if err != nil {
				http.Error(w, fmt.Sprintf("Error importing channels: %v", err), http.StatusBadRequest)
				return
			}

		case "talkgroups":
			if overwrite {
				database.DB.Where("type = ?", models.ContactTypeGroup).Delete(&models.Contact{})
			}

			var contacts []models.Contact
			var err error

			switch radioPlatform {
			case "dm32uv":
				err = importer.ImportDM32UVTalkgroups(database.DB, f)
			case "at890":
				err = importer.ImportAnyTone890Talkgroups(database.DB, f)
			default:
				contacts, err = importer.ImportGenericTalkgroups(f)
				if err == nil {
					for _, c := range contacts {
						if err := database.DB.Where("dmr_id = ? AND type = ?", c.DMRID, c.Type).FirstOrCreate(&c).Error; err == nil {
							count++
						}
					}
				}
			}

			if err != nil {
				http.Error(w, fmt.Sprintf("Error importing talkgroups: %v", err), http.StatusBadRequest)
				return
			}

		case "contacts":
			if overwrite {
				database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.DigitalContact{})
			}

			var err error
			switch radioPlatform {
			case "dm32uv":
				err = importer.ImportDM32UVDigitalContacts(database.DB, f)
			case "at890":
				err = importer.ImportAnyTone890DigitalContacts(database.DB, f)
			default:
				f.Seek(0, 0)
				contacts, rErr := importer.ImportRadioIDCSV(f, nil)
				if rErr != nil {
					err = rErr
				} else {
					batchSize := 1000
					for i := 0; i < len(contacts); i += batchSize {
						end := i + batchSize
						if end > len(contacts) {
							end = len(contacts)
						}
						batch := contacts[i:end]
						database.DB.Clauses(clause.OnConflict{
							Columns:   []clause.Column{{Name: "dmr_id"}},
							DoUpdates: clause.AssignmentColumns([]string{"name", "callsign", "city", "state", "country", "remarks"}),
						}).Create(&batch)
						count += len(batch)
					}
				}
			}

			if err != nil {
				http.Error(w, fmt.Sprintf("Error importing digital contacts: %v", err), http.StatusBadRequest)
				return
			}

		case "zones":
			var err error
			switch radioPlatform {
			case "dm32uv":
				err = importer.ImportDM32UVZones(database.DB, f)
			case "at890":
				err = importer.ImportAnyTone890Zones(database.DB, f)
			default:
				http.Error(w, "Generic Zone import not supported yet", http.StatusBadRequest)
				return
			}
			if err != nil {
				http.Error(w, fmt.Sprintf("Error importing zones: %v", err), http.StatusBadRequest)
				return
			}
		}

		CurrentProgress.mu.Lock()
		CurrentProgress.Status = "completed"
		CurrentProgress.Message = fmt.Sprintf("Imported %s successfully.", importType)
		CurrentProgress.Processed = count
		CurrentProgress.mu.Unlock()
		BroadcastProgress()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("Successfully imported %s", importType),
			"count":   count,
		})
		return
	}

	if format == "radioid" {
		sourceMode := r.FormValue("source_mode")
		overwrite := r.FormValue("overwrite") == "true"
		if overwrite {
			database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.DigitalContact{})
		}

		var activeIDs map[int]bool
		filterFile, _, err := r.FormFile("filter_file")
		if err == nil {
			defer filterFile.Close()
			ids, err := importer.ParseBrandmeisterLastHeard(filterFile)
			if err == nil {
				activeIDs = ids
			}
		}

		var reader io.Reader

		CurrentProgress.mu.Lock()
		CurrentProgress.Total = 0
		CurrentProgress.Processed = 0
		CurrentProgress.Status = "running"
		CurrentProgress.Message = "Initializing..."
		CurrentProgress.mu.Unlock()
		BroadcastProgress()

		if sourceMode == "download" {
			CurrentProgress.mu.Lock()
			CurrentProgress.Message = "Downloading contacts from RadioID.net..."
			CurrentProgress.mu.Unlock()
			BroadcastProgress()

			resp, err := http.Get("https://database.radioid.net/static/user.csv")
			if err != nil {
				http.Error(w, fmt.Sprintf("Error downloading from RadioID: %v", err), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			reader = resp.Body
		} else {
			fileRef, _ := os.Open(path)
			defer fileRef.Close()
			reader = fileRef
		}

		CurrentProgress.mu.Lock()
		CurrentProgress.Message = "Parsing CSV data..."
		CurrentProgress.mu.Unlock()
		BroadcastProgress()

		contacts, err := importer.ImportRadioIDCSV(reader, activeIDs)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing RadioID CSV: %v", err), http.StatusBadRequest)
			return
		}

		CurrentProgress.mu.Lock()
		CurrentProgress.Total = len(contacts)
		CurrentProgress.Processed = 0
		CurrentProgress.Status = "running"
		CurrentProgress.Message = "Starting import..."
		CurrentProgress.mu.Unlock()
		BroadcastProgress()

		err = database.DB.Transaction(func(tx *gorm.DB) error {
			if len(contacts) > 0 {
				batchSize := 1000
				for i := 0; i < len(contacts); i += batchSize {
					end := i + batchSize
					if end > len(contacts) {
						end = len(contacts)
					}

					CurrentProgress.mu.Lock()
					CurrentProgress.Processed = i
					CurrentProgress.Message = fmt.Sprintf("Importing contacts %d to %d...", i, end)
					CurrentProgress.mu.Unlock()
					BroadcastProgress()

					batch := contacts[i:end]
					if err := tx.Clauses(clause.OnConflict{
						Columns: []clause.Column{{Name: "dmr_id"}},
						DoUpdates: clause.AssignmentColumns([]string{
							"name", "callsign", "city", "state", "country", "remarks",
							"deleted_at", "updated_at",
						}),
					}).Create(&batch).Error; err != nil {
						return err
					}
				}
			}
			return nil
		})

		CurrentProgress.mu.Lock()
		if err != nil {
			CurrentProgress.Status = "error"
			CurrentProgress.Message = fmt.Sprintf("Error: %v", err)
		} else {
			CurrentProgress.Processed = len(contacts)
			CurrentProgress.Status = "completed"
			CurrentProgress.Message = fmt.Sprintf("Imported %d contacts successfully.", len(contacts))
		}
		CurrentProgress.mu.Unlock()
		BroadcastProgress()

		if err != nil {
			http.Error(w, fmt.Sprintf("Error saving contacts: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"imported": len(contacts),
			"skipped":  0,
			"message":  fmt.Sprintf("Processed %d contacts successfully.", len(contacts)),
		})
		return
	}

	if format == "filter_list" {
		listName := r.FormValue("list_name")
		if listName == "" {
			http.Error(w, "List name is required", http.StatusBadRequest)
			return
		}

		if err := importer.ImportFilterListToDB(database.DB, path, listName); err != nil {
			http.Error(w, fmt.Sprintf("Error importing filter list: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("Successfully imported filter list '%s'", listName),
		})
		return
	}

	if path == "" {
		http.Error(w, "File is required for generic import", http.StatusBadRequest)
		return
	}

	overwrite := r.FormValue("overwrite") == "true"
	if overwrite {
		database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Channel{})
		database.DB.Exec("DELETE FROM sqlite_sequence WHERE name = 'channels'")
	}

	var channels []models.Channel

	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "Error opening uploaded file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	headerBuf := make([]byte, 1024)
	n, _ := f.Read(headerBuf)
	headerStr := string(headerBuf[:n])
	f.Seek(0, 0)

	if strings.Contains(headerStr, "Location") && strings.Contains(headerStr, "CrossMode") {
		channels, err = importer.ImportChirpCSV(f)
	} else {
		channels, err = importer.ImportChannelsCSV(f)
		if err != nil || len(channels) == 0 {
			f.Seek(0, 0)
			chirpChannels, chirpErr := importer.ImportChirpCSV(f)
			if chirpErr == nil && len(chirpChannels) > 0 {
				channels = chirpChannels
				err = nil
			}
		}
	}

	services.ResolveContacts(database.DB, channels)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing CSV: %v", err), http.StatusBadRequest)
		return
	}

	count := 0
	skipped := 0
	for _, ch := range channels {
		if !overwrite {
			var existing models.Channel
			if database.DB.Where("name = ? AND rx_frequency = ?", ch.Name, ch.RxFrequency).First(&existing).Error == nil {
				skipped++
				continue
			}
		}
		if result := database.DB.Create(&ch); result.Error == nil {
			count++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"imported": count,
		"skipped":  skipped,
	})
}

func HandleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := r.URL.Query().Get("format")
	radio := r.URL.Query().Get("radio")

	zoneIDsStr := r.URL.Query()["zone_id"]
	var zoneIDs []int

	for _, idStr := range zoneIDsStr {
		parts := strings.Split(idStr, ",")
		for _, p := range parts {
			if id, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				zoneIDs = append(zoneIDs, id)
			}
		}
	}

	var filterListID uint
	useList := r.URL.Query().Get("use_list")
	if useList != "" {
		var list models.ContactList
		if err := database.DB.Where("name = ?", useList).First(&list).Error; err == nil {
			filterListID = list.ID
		} else {
			fmt.Printf("Warning: Filter list '%s' not found.\n", useList)
		}
	}

	if (format == "" || format == "zip") && radio != "" {
		format = radio
	}

	switch format {
	case "db":
		filename := "codeplugs.db"
		w.Header().Set("Content-Type", "application/x-sqlite3")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		http.ServeFile(w, r, filename)
		return

	case "dm32uv", "at890":
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"codeplug_%s.zip\"", format))

		zipWriter := zip.NewWriter(w)
		defer zipWriter.Close()

		switch format {
		case "dm32uv":
			var channels []models.Channel
			db := database.DB.Model(&models.Channel{}).Preload("Contact").Where("skip = ?", false)

			if len(zoneIDs) > 0 {
				db = db.Joins("JOIN zone_channels ON zone_channels.channel_id = channels.id").
					Where("zone_channels.zone_id IN ?", zoneIDs)
			}

			db.Find(&channels)

			f, _ := zipWriter.Create("channels.csv")
			exporter.ExportDM32UVChannels(channels, f)

			var zones []models.Zone
			zdb := database.DB.Preload("Channels")
			if len(zoneIDs) > 0 {
				zdb = zdb.Where("id IN ?", zoneIDs)
			}
			zdb.Find(&zones)

			f, _ = zipWriter.Create("zones.csv")
			exporter.ExportDM32UVZones(zones, f)

			var talkgroups []models.Contact
			database.DB.Where("type = ?", models.ContactTypeGroup).Find(&talkgroups)
			f, _ = zipWriter.Create("talkgroups.csv")
			exporter.ExportDM32UVTalkgroups(talkgroups, f)

			var digitalContacts []models.DigitalContact
			query := database.DB.Model(&models.DigitalContact{})

			if filterListID > 0 {
				query = query.Where("dmr_id IN (?)", database.DB.Model(&models.ContactListEntry{}).Select("dmr_id").Where("contact_list_id = ?", filterListID))
			} else {
				query = query.Limit(50000)
			}

			query.Find(&digitalContacts)
			f, _ = zipWriter.Create("digital_contacts.csv")
			exporter.ExportDM32UVDigitalContacts(digitalContacts, f)

		case "at890":
			tempDir, err := os.MkdirTemp("", "at890_export_*")
			if err != nil {
				http.Error(w, "Failed to create temp dir", http.StatusInternalServerError)
				return
			}
			defer os.RemoveAll(tempDir)

			if err := exporter.ExportAnyTone890(database.DB, tempDir, filterListID); err != nil {
				http.Error(w, "Failed to export 890", http.StatusInternalServerError)
				return
			}

			err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}

				relPath, _ := filepath.Rel(tempDir, path)
				f, err := zipWriter.Create(relPath)
				if err != nil {
					return err
				}

				content, _ := os.ReadFile(path)
				f.Write(content)
				return nil
			})
		}
		return
	}

	w.Header().Set("Content-Type", "text/csv")

	filename := "codeplug.csv"
	if format == "chirp" {
		filename = "chirp_export.csv"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	var channels []models.Channel
	query := database.DB.Model(&models.Channel{}).Preload("Contact").Where("skip = ?", false)

	if len(zoneIDs) > 0 {
		query = query.Joins("JOIN zone_channels ON zone_channels.channel_id = channels.id").
			Where("zone_channels.zone_id IN ?", zoneIDs)
	}

	query.Find(&channels)

	if format == "chirp" {
		exporter.ExportChirpCSV(channels, w)
	} else {
		exporter.ExportDB25D(channels, w, false)
	}
}

func HandleContacts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		source := r.URL.Query().Get("source")

		if source == "RadioID" {
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit < 1 {
				limit = 50
			}
			search := r.URL.Query().Get("search")
			sort := r.URL.Query().Get("sort")
			order := r.URL.Query().Get("order")

			offset := (page - 1) * limit

			var contacts []models.DigitalContact
			var total int64

			db := database.DB.Model(&models.DigitalContact{})

			if search != "" {
				term := "%" + search + "%"
				db = db.Where("name LIKE ? OR callsign LIKE ? OR CAST(dmr_id AS TEXT) LIKE ?", term, term, term)
			}

			db.Count(&total)

			if sort != "" {
				if order != "desc" {
					order = "asc"
				}
				db = db.Order(fmt.Sprintf("%s %s", sort, order))
			} else {
				db = db.Order("id asc")
			}

			db.Limit(limit).Offset(offset).Find(&contacts)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": contacts,
				"meta": map[string]interface{}{
					"total": total,
					"page":  page,
					"limit": limit,
				},
			})
			return
		}

		var contacts []models.Contact
		database.DB.Find(&contacts)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": contacts,
		})

	case "POST":
		var c models.Contact
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if c.ID == 0 {
			database.DB.Create(&c)
		} else {
			database.DB.Save(&c)
		}
		json.NewEncoder(w).Encode(c)
	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			var count int64
			database.DB.Model(&models.Channel{}).Where("contact_id = ?", id).Count(&count)
			if count > 0 {
				http.Error(w, "Contact is in use by channels", http.StatusConflict)
				return
			}
			database.DB.Delete(&models.Contact{}, id)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func HandleZones(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		id := r.URL.Query().Get("id")
		if id != "" {
			var zone models.Zone
			// Preload via ZoneChannels to guarantee order
			if err := database.DB.Preload("ZoneChannels", func(db *gorm.DB) *gorm.DB {
				return db.Order("sort_order ASC")
			}).Preload("ZoneChannels.Channel").First(&zone, id).Error; err != nil {
				http.Error(w, "Zone not found", http.StatusNotFound)
				return
			}
			// Map ZoneChannels back to Channels for JSON compatibility (or use ZoneChannels in frontend?)
			// Maintaining compatibility with Channels field in JSON:
			zone.Channels = make([]models.Channel, len(zone.ZoneChannels))
			for i, zc := range zone.ZoneChannels {
				zone.Channels[i] = zc.Channel
			}
			json.NewEncoder(w).Encode(zone)
			return
		}

		var zones []models.Zone
		database.DB.Preload("ZoneChannels", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).Preload("ZoneChannels.Channel").Find(&zones)

		for i := range zones {
			zones[i].Channels = make([]models.Channel, len(zones[i].ZoneChannels))
			for j, zc := range zones[i].ZoneChannels {
				zones[i].Channels[j] = zc.Channel
			}
		}
		json.NewEncoder(w).Encode(zones)
	case "POST":
		var z models.Zone
		if err := json.NewDecoder(r.Body).Decode(&z); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if z.ID == 0 {
			database.DB.Create(&z)
		} else {
			if err := database.DB.Model(&z).Where("id = ?", z.ID).Update("name", z.Name).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		json.NewEncoder(w).Encode(z)
	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			database.DB.Exec("DELETE FROM zone_channels WHERE zone_id = ?", id)
			database.DB.Delete(&models.Zone{}, id)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func HandleZoneAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Zone ID required", http.StatusBadRequest)
		return
	}
	zoneID, _ := strconv.Atoi(id)

	var channelIDs []int
	if err := json.NewDecoder(r.Body).Decode(&channelIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Manual transaction to update zone_channels with order
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Clear existing
	if err := tx.Exec("DELETE FROM zone_channels WHERE zone_id = ?", zoneID).Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to clear existing", http.StatusInternalServerError)
		return
	}

	// 2. Insert new with order
	// Batch insert not easily done via GORM Association mode with custom join fields unless we use SetupJoinTable + Create on the Join Model directly.
	if len(channelIDs) > 0 {
		var zcs []models.ZoneChannel
		for i, cid := range channelIDs {
			zcs = append(zcs, models.ZoneChannel{
				ZoneID:    uint(zoneID),
				ChannelID: uint(cid),
				SortOrder: i + 1,
			})
		}
		if err := tx.Create(&zcs).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to assign channels", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Commit failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleScanLists(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		var lists []models.ScanList
		database.DB.Preload("Channels").Find(&lists)
		json.NewEncoder(w).Encode(lists)
	case "POST":
		var list models.ScanList
		if err := json.NewDecoder(r.Body).Decode(&list); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if list.ID == 0 {
			database.DB.Create(&list)
		} else {
			database.DB.Model(&list).Update("name", list.Name)
		}
		json.NewEncoder(w).Encode(list)
	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			database.DB.Exec("DELETE FROM scan_list_channels WHERE scan_list_id = ?", id)
			database.DB.Delete(&models.ScanList{}, id)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func HandleScanListAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ScanListID int   `json:"scan_list_id"`
		ChannelIDs []int `json:"channel_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var list models.ScanList
	if err := database.DB.First(&list, req.ScanListID).Error; err != nil {
		http.Error(w, "Scan List not found", http.StatusNotFound)
		return
	}

	var channels []models.Channel
	database.DB.Find(&channels, req.ChannelIDs)

	if err := database.DB.Model(&list).Association("Channels").Replace(&channels); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleFilterLists(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		id := r.URL.Query().Get("id")

		if id != "" {
			var list models.ContactList
			if err := database.DB.First(&list, id).Error; err != nil {
				http.Error(w, "List not found", http.StatusNotFound)
				return
			}

			if r.URL.Query().Get("mode") == "ids" {
				var ids []int
				database.DB.Model(&models.ContactListEntry{}).Where("contact_list_id = ?", list.ID).Pluck("dmr_id", &ids)
				json.NewEncoder(w).Encode(ids)
				return
			}

			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit < 1 {
				limit = 100
			}
			offset := (page - 1) * limit
			search := r.URL.Query().Get("search")

			query := database.DB.Model(&models.ContactListEntry{}).Where("contact_list_id = ?", list.ID)

			if search != "" {
				query = query.Where("CAST(dmr_id AS TEXT) LIKE ?", "%"+search+"%")
			}

			var total int64
			query.Count(&total)

			var entries []models.ContactListEntry
			query.Limit(limit).Offset(offset).Find(&entries)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": entries,
				"meta": map[string]interface{}{
					"total": total,
					"page":  page,
					"limit": limit,
				},
			})
			return
		}

		var lists []models.ContactList
		database.DB.Find(&lists)
		json.NewEncoder(w).Encode(lists)
	}
}
