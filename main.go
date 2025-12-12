package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"codeplugs/database"
	"codeplugs/exporter"
	"codeplugs/importer"
	"codeplugs/models"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed frontend/dist
var frontendDist embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocketHub maintains the set of active clients and broadcasts messages
type WebSocketHub struct {
	// Registered clients.
	clients map[*websocket.Conn]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *websocket.Conn

	// Unregister requests from clients.
	unregister chan *websocket.Conn

	mu sync.Mutex
}

func newHub() *WebSocketHub {
	return &WebSocketHub{
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		clients:    make(map[*websocket.Conn]bool),
	}
}

func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

var hub = newHub()

// ImportProgress tracks the status of a running import
type ImportProgress struct {
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Status    string `json:"status"` // "running", "completed", "error"
	Message   string `json:"message"`
	mu        sync.Mutex
}

var currentProgress = &ImportProgress{Status: "idle"}

func broadcastProgress() {
	currentProgress.mu.Lock()
	defer currentProgress.mu.Unlock()

	msg, _ := json.Marshal(map[string]interface{}{
		"type": "import_progress",
		"data": currentProgress,
	})
	hub.broadcast <- msg
}

func main() {
	dbPath := flag.String("db", "codeplugs.db", "Path to SQLite database")
	importFile := flag.String("import", "", "Path to CSV file to import")
	exportFile := flag.String("export", "", "Path to CSV file to export to")
	format := flag.String("format", "db25d", "Export format: db25d, chirp")
	serve := flag.Bool("serve", false, "Start Web UI server")
	port := flag.String("port", "8080", "Port for Web UI server")
	zoneName := flag.String("zone", "", "Zone name to assign imported channels to or filter export by")

	// Additional flags
	radio := flag.String("radio", "db25d", "Unknown/Target radio profile: db25d, dm32uv, at890")
	filterList := flag.String("filter-list", "", "Path to CSV/Text file containing allowed DMR IDs for contact export")
	limit := flag.Int("limit", 0, "Limit number of contacts exported (0 = no limit, or default for radio)")

	flag.Parse()

	database.Connect(*dbPath)

	// Start WebSocket Hub
	go hub.run()

	if *serve {
		startServer(*port)
		return
	}

	if *importFile != "" {
		if *radio == "dm32uv" {
			info, err := os.Stat(*importFile)
			if err != nil {
				log.Fatalf("Error stating import path: %v", err)
			}

			if info.IsDir() {
				fmt.Printf("Importing DM32UV from directory %s...\n", *importFile)
				// 1. Digital Contacts
				if f, err := os.Open(filepath.Join(*importFile, "digital_contacts.csv")); err == nil {
					fmt.Println("Importing Digital Contacts...")
					if err := importer.ImportDM32UVDigitalContacts(database.DB, f); err != nil {
						log.Printf("Error importing digital contacts: %v", err)
					}
					f.Close()
				}
				// 2. Talkgroups
				if f, err := os.Open(filepath.Join(*importFile, "talkgroups.csv")); err == nil {
					fmt.Println("Importing Talkgroups...")
					if err := importer.ImportDM32UVTalkgroups(database.DB, f); err != nil {
						log.Printf("Error importing talkgroups: %v", err)
					}
					f.Close()
				}
				// 3. Channels
				if f, err := os.Open(filepath.Join(*importFile, "channels.csv")); err == nil {
					fmt.Println("Importing Channels...")
					if err := importer.ImportDM32UVChannels(database.DB, f); err != nil {
						log.Printf("Error importing channels: %v", err)
					}
					f.Close()
				}
				// 4. Zones
				if f, err := os.Open(filepath.Join(*importFile, "zones.csv")); err == nil {
					fmt.Println("Importing Zones...")
					if err := importer.ImportDM32UVZones(database.DB, f); err != nil {
						log.Printf("Error importing zones: %v", err)
					}
					f.Close()
				}
			} else {
				fmt.Printf("Importing DM32UV Channels from %s...\n", *importFile)
				f, err := os.Open(*importFile)
				if err != nil {
					log.Fatalf("Error opening file: %v", err)
				}
				defer f.Close()

				if err := importer.ImportDM32UVChannels(database.DB, f); err != nil {
					log.Fatalf("Error importing channels: %v", err)
				}
			}
			fmt.Println("Import complete.")
			return
		} else if *radio == "at890" {
			fmt.Printf("Importing AnyTone 890 from %s...\n", *importFile)
			// Open file for reading
			f, err := os.Open(*importFile)
			if err != nil {
				log.Fatalf("Error opening file: %v", err)
			}
			defer f.Close()

			if err := importer.ImportAnyTone890Channels(database.DB, f); err != nil {
				log.Fatalf("Error importing channels: %v", err)
			}
			fmt.Println("Import complete.")
			return
		}

		// Generic Import (DB25-D / Chirp) to memory first
		var channels []models.Channel
		var err error

		// Open the file
		f, err := os.Open(*importFile)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		defer f.Close()

		fmt.Printf("Importing from %s...\n", *importFile)
		channels, err = importer.ImportChannelsCSV(f)
		// Fallback to Chirp if Generic failed or returned empty
		if err != nil || len(channels) == 0 {
			_, seekErr := f.Seek(0, 0)
			if seekErr == nil {
				chirpChannels, chirpErr := importer.ImportChirpCSV(f)
				if chirpErr == nil && len(chirpChannels) > 0 {
					channels = chirpChannels
					err = nil
				}
			}
		}

		if err != nil {
			log.Fatalf("Error importing CSV: %v", err)
		}

		count := 0
		skipped := 0

		var zone *models.Zone
		if *zoneName != "" {
			var err error
			zone, err = models.FindOrCreateZone(database.DB, *zoneName)
			if err != nil {
				log.Fatalf("Error finding/creating zone: %v", err)
			}
		}

		for _, ch := range channels {
			var existing models.Channel
			if err := database.DB.Where("name = ? AND rx_frequency = ?", ch.Name, ch.RxFrequency).First(&existing).Error; err == nil {
				skipped++
				continue
			}

			result := database.DB.Create(&ch)
			if result.Error != nil {
				log.Printf("Failed to save channel %s: %v", ch.Name, result.Error)
			} else {
				if zone != nil {
					if err := database.DB.Model(zone).Association("Channels").Append(&ch); err != nil {
						log.Printf("Failed to add channel %s to zone %s: %v", ch.Name, zone.Name, err)
					}
				}
				count++
			}
		}
		fmt.Printf("Imported %d channels (skipped %d duplicates).\n", count, skipped)

	} else if *exportFile != "" {
		if *radio == "dm32uv" {
			// DM32UV Export Logic
			baseFilename := *exportFile
			// Remove .csv extension if present to append suffixes
			if len(baseFilename) > 4 && baseFilename[len(baseFilename)-4:] == ".csv" {
				baseFilename = baseFilename[:len(baseFilename)-4]
			}

			fmt.Printf("Exporting for DM32UV to %s-*.csv ...\n", baseFilename)

			// 1. Export Channels
			var channels []models.Channel
			query := database.DB.Model(&models.Channel{}).Preload("Contact").Where("skip = ?", false)
			if *zoneName != "" {
				var zone models.Zone
				if err := database.DB.Where("name = ?", *zoneName).First(&zone).Error; err != nil {
					log.Fatalf("Zone not found: %s", *zoneName)
				}
				query = query.Joins("JOIN zone_channels ON zone_channels.channel_id = channels.id").
					Where("zone_channels.zone_id = ?", zone.ID)
			}
			query.Find(&channels)

			chanFile, err := os.Create(baseFilename + "_channels.csv")
			if err != nil {
				log.Fatalf("Error creating channels file: %v", err)
			}
			defer chanFile.Close()
			err = exporter.ExportDM32UVChannels(channels, chanFile)
			if err != nil {
				log.Fatalf("Error exporting channels: %v", err)
			}
			fmt.Printf(" - Channels: %s_channels.csv\n", baseFilename)

			// 2. Export Zones
			var zones []models.Zone
			database.DB.Preload("Channels").Find(&zones)
			if *zoneName != "" {
				// Filter to only this zone
				filteredZones := []models.Zone{}
				for _, z := range zones {
					if z.Name == *zoneName {
						filteredZones = append(filteredZones, z)
						break
					}
				}
				zones = filteredZones
			}

			zoneFile, err := os.Create(baseFilename + "_zones.csv")
			if err != nil {
				log.Fatalf("Error creating zones file: %v", err)
			}
			defer zoneFile.Close()
			err = exporter.ExportDM32UVZones(zones, zoneFile)
			if err != nil {
				log.Fatalf("Error exporting zones: %v", err)
			}
			fmt.Printf(" - Zones: %s_zones.csv\n", baseFilename)

			// 3. Export Talk Groups (Local Contacts)
			var talkgroups []models.Contact
			database.DB.Where("type = ?", models.ContactTypeGroup).Find(&talkgroups)

			tgFile, err := os.Create(baseFilename + "_talkgroups.csv")
			if err != nil {
				log.Fatalf("Error creating talkgroups file: %v", err)
			}
			defer tgFile.Close()
			err = exporter.ExportDM32UVTalkgroups(talkgroups, tgFile)
			if err != nil {
				log.Fatalf("Error exporting talkgroups: %v", err)
			}
			fmt.Printf(" - TalkGroups: %s_talkgroups.csv\n", baseFilename)

			// 4. Export Digital Contacts (CSV Contacts)
			// Filter logic
			var allowedIDs map[int]bool
			if *filterList != "" {
				var err error
				allowedIDs, err = importer.LoadFilterList(*filterList)
				if err != nil {
					log.Fatalf("Error loading filter list: %v", err)
				}
				fmt.Printf("Loaded %d allowed IDs from filter list.\n", len(allowedIDs))
			}

			// Fetch Digital Contacts
			var digitalContacts []models.DigitalContact
			// We can filter in DB if we want, but if allowedIDs is a map, we might need to iterate or fetch all then filter.
			// Ideally we fetch all and filter in memory if not too huge, or use WHERE IN (chunked).
			// Given simple implementation: fetch all is dangerous if huge?
			// But for now, let's fetch all. DB is local.
			database.DB.Find(&digitalContacts)

			filteredContacts := []models.DigitalContact{}
			for _, c := range digitalContacts {
				if allowedIDs != nil {
					if !allowedIDs[c.DMRID] {
						continue
					}
				}
				filteredContacts = append(filteredContacts, c)
			}

			// Apply Limit
			maxContacts := 50000 // Default for DM32UV
			if *limit > 0 {
				maxContacts = *limit
			}

			if len(filteredContacts) > maxContacts {
				fmt.Printf("Warning: Contact count %d exceeds limit %d. Truncating.\n", len(filteredContacts), maxContacts)
				filteredContacts = filteredContacts[:maxContacts]
			}

			dcFile, err := os.Create(baseFilename + "_digital_contacts.csv")
			if err != nil {
				log.Fatalf("Error creating digital contacts file: %v", err)
			}
			defer dcFile.Close()
			err = exporter.ExportDM32UVDigitalContacts(filteredContacts, dcFile)
			if err != nil {
				log.Fatalf("Error exporting digital contacts: %v", err)
			}
			fmt.Printf(" - Digital Contacts: %s_digital_contacts.csv (%d records)\n", baseFilename, len(filteredContacts))

		} else if *radio == "at890" {
			// AnyTone 890 Export Logic
			fmt.Printf("Exporting AnyTone 890 to directory %s...\n", *exportFile)
			if err := exporter.ExportAnyTone890(database.DB, *exportFile); err != nil {
				log.Fatalf("Error exporting to AnyTone 890: %v", err)
			}
			fmt.Println("Export complete.")
			return
		} else {
			// Existing DB25-D Export
			fmt.Printf("Exporting to %s (format: %s)...\n", *exportFile, *format)
			var channels []models.Channel
			query := database.DB.Model(&models.Channel{}).Preload("Contact").Where("skip = ?", false)

			if *zoneName != "" {
				var zone models.Zone
				if err := database.DB.Where("name = ?", *zoneName).First(&zone).Error; err != nil {
					log.Fatalf("Zone not found: %s", *zoneName)
				}
				query = query.Joins("JOIN zone_channels ON zone_channels.channel_id = channels.id").
					Where("zone_channels.zone_id = ?", zone.ID)
			}

			query.Find(&channels)

			var err error

			f, err := os.Create(*exportFile)
			if err != nil {
				log.Fatalf("Error creating export file: %v", err)
			}
			defer f.Close()

			if *format == "chirp" {
				err = exporter.ExportChirpCSV(channels, f)
			} else {
				exporter.ExportDB25D(channels, f, false)
			}

			if err != nil {
				log.Fatalf("Error exporting CSV: %v", err)
			}
			fmt.Printf("Exported %d channels to %s.\n", len(channels), *exportFile)
		}
	} else {
		var channelCount int64
		database.DB.Model(&models.Channel{}).Count(&channelCount)
		fmt.Printf("Database contains %d channels.\n", channelCount)
	}
}

func startServer(port string) {
	// API Routes
	http.HandleFunc("/api/channels", handleChannels)
	http.HandleFunc("/api/import", handleImport)
	http.HandleFunc("/api/export", handleExport)
	http.HandleFunc("/api/contacts", handleContacts)
	http.HandleFunc("/api/zones", handleZones)
	http.HandleFunc("/api/zones/assign", handleZoneAssignment)
	http.HandleFunc("/api/ws", handleWebSocket)

	// Static Files
	distFS, err := fs.Sub(frontendDist, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", http.FileServer(http.FS(distFS)))

	fmt.Printf("Starting server on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var channels []models.Channel
		database.DB.Find(&channels)
		json.NewEncoder(w).Encode(channels)
	} else if r.Method == "POST" {
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
	} else if r.Method == "DELETE" {
		// Handle delete... need ID from URL or body
		// Simplified: assume ID in body for now or query param
		id := r.URL.Query().Get("id")
		if id != "" {
			database.DB.Delete(&models.Channel{}, id)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// limit to 100MB (RadioID CSV is large)
	r.ParseMultipartForm(100 << 20)

	format := r.FormValue("format")
	sourceMode := r.FormValue("source_mode")

	// Check for file UNLESS we are in download mode
	var path string
	var tempFile *os.File
	var err error

	if sourceMode != "download" {
		if format == "zip" {
			// Handle Zip Import
			file, _, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Error retrieving file", http.StatusBadRequest)
				return
			}
			defer file.Close()

			// Helper to read zip content
			// We need to read the whole file into a ReaderAt
			// Since FormFile gives us a multipart.File which works as a Reader, ensure we can convert or read all.
			// Simply Copy to bytes buffer.
			buf := new(bytes.Buffer)
			io.Copy(buf, file)
			bytesReader := bytes.NewReader(buf.Bytes())

			zipReader, err := zip.NewReader(bytesReader, int64(buf.Len()))
			if err != nil {
				http.Error(w, "Invalid zip file", http.StatusBadRequest)
				return
			}

			// Import Order: Digital Contacts -> Talkgroups -> Channels -> Zones
			// We iterate or find specific files.
			filesMap := make(map[string]*zip.File)
			for _, f := range zipReader.File {
				filesMap[f.Name] = f
			}

			// 1. Digital Contacts
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

			// 2. Talkgroups
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

			// 3. Channels
			if f, ok := filesMap["channels.csv"]; ok {
				rc, _ := f.Open()
				err := importer.ImportDM32UVChannels(database.DB, rc)
				rc.Close()
				if err != nil {
					log.Printf("Error importing channels: %v", err)
					http.Error(w, fmt.Sprintf("Error importing channels: %v", err), http.StatusBadRequest)
					return
				}
				// Skip resolveContacts and manual save loop as importer handles it
			}

			// 4. Zones
			if f, ok := filesMap["zones.csv"]; ok {
				rc, _ := f.Open()
				err := importer.ImportDM32UVZones(database.DB, rc)
				rc.Close()
				if err != nil {
					log.Printf("Error importing zones: %v", err)
					http.Error(w, fmt.Sprintf("Error importing zones: %v", err), http.StatusBadRequest)
					return
				}
				// Skip manual save
			}

			fmt.Fprintf(w, "Zip Import Complete")
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Error retrieving file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Save temp file
		tempFile, err = os.CreateTemp("", "upload-*.csv")
		if err != nil {
			http.Error(w, "Error creating temp file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(tempFile.Name()) // Clean up after handler
		defer tempFile.Close()

		if _, err := io.Copy(tempFile, file); err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}
		path = tempFile.Name()
	}

	if format == "radioid" {
		// Digital Contact Import

		// Overwrite?
		overwrite := r.FormValue("overwrite") == "true"
		if overwrite {
			// Delete existing RadioID contacts
			// Since we moved to DigitalContact table, we can just clear it or filter if we had a source column (but we don't need one now)
			database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.DigitalContact{})
		}

		// Filter File?
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

		if sourceMode == "download" {
			// Download from RadioID.net
			fmt.Println("Downloading contacts from RadioID.net...")
			resp, err := http.Get("https://database.radioid.net/static/user.csv")
			if err != nil {
				http.Error(w, fmt.Sprintf("Error downloading from RadioID: %v", err), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			reader = resp.Body
		} else {
			// Use uploaded file
			fileRef, _ := os.Open(path)
			defer fileRef.Close()
			reader = fileRef
		}

		contacts, err := importer.ImportRadioIDCSV(reader, activeIDs)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing RadioID CSV: %v", err), http.StatusBadRequest)
			return
		}

		// Batch Insert with Conflict Handling
		// We use a transaction to ensure speed and integrity

		// Reset Progress
		currentProgress.mu.Lock()
		currentProgress.Total = len(contacts)
		currentProgress.Processed = 0
		currentProgress.Status = "running"
		currentProgress.Message = "Starting import..."
		currentProgress.mu.Unlock()
		broadcastProgress()

		err = database.DB.Transaction(func(tx *gorm.DB) error {
			if len(contacts) > 0 {
				batchSize := 1000
				for i := 0; i < len(contacts); i += batchSize {
					end := i + batchSize
					if end > len(contacts) {
						end = len(contacts)
					}

					// Update Progress
					currentProgress.mu.Lock()
					currentProgress.Processed = i
					currentProgress.Message = fmt.Sprintf("Importing contacts %d to %d...", i, end)
					currentProgress.mu.Unlock()
					broadcastProgress()

					batch := contacts[i:end]
					if err := tx.Clauses(clause.OnConflict{
						Columns: []clause.Column{{Name: "dmr_id"}}, // DigitalContact unique index is DMRID (and deleted_at for soft delete, but uniqueness is primarily DMRID)
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

		currentProgress.mu.Lock()
		if err != nil {
			currentProgress.Status = "error"
			currentProgress.Message = fmt.Sprintf("Error: %v", err)
		} else {
			currentProgress.Processed = len(contacts)
			currentProgress.Status = "completed"
			currentProgress.Message = fmt.Sprintf("Imported %d contacts successfully.", len(contacts))
		}
		currentProgress.mu.Unlock()
		broadcastProgress()

		if err != nil {
			http.Error(w, fmt.Sprintf("Error saving contacts: %v", err), http.StatusInternalServerError)
			return
		}

		// Calculate stats (approximate since we did batch insert)
		// Basic math: total imported = attempts. True "new" count is hard to get from simple batch insert without return
		// But for user feedback, we can say "Processed X contacts".
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"imported": len(contacts),
			"skipped":  0, // We don't track skips efficiently in batch mode
			"message":  fmt.Sprintf("Processed %d contacts successfully.", len(contacts)),
		})
		return
	}

	// Normal Channel Import
	if path == "" {
		http.Error(w, "File is required for generic import", http.StatusBadRequest)
		return
	}

	var channels []models.Channel

	// Open temp file for reading
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "Error opening uploaded file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	channels, err = importer.ImportChannelsCSV(f)
	if err != nil || len(channels) == 0 {
		f.Seek(0, 0)
		chirpChannels, chirpErr := importer.ImportChirpCSV(f)
		if chirpErr == nil && len(chirpChannels) > 0 {
			channels = chirpChannels
			err = nil
		}
	}

	resolveContacts(channels)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing CSV: %v", err), http.StatusBadRequest)
		return
	}

	count := 0
	skipped := 0
	for _, ch := range channels {
		// Simple deduplication
		var existing models.Channel
		if err := database.DB.Where("name = ? AND rx_frequency = ?", ch.Name, ch.RxFrequency).First(&existing).Error; err == nil {
			skipped++
			continue
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

func handleExport(w http.ResponseWriter, r *http.Request) {
	// Generate CSV and serve as download
	format := r.URL.Query().Get("format")
	radio := r.URL.Query().Get("radio")

	if format == "" {
		if radio == "dm32uv" {
			format = "zip"
		} else {
			format = "db25d"
		}
	}

	var channels []models.Channel
	query := database.DB.Model(&models.Channel{}).Preload("Contact").Where("skip = ?", false)

	// Zone Filtering
	zoneName := r.URL.Query().Get("zone")
	if zoneName != "" {
		// Verify zone exists
		var z models.Zone
		if err := database.DB.Where("name = ?", zoneName).First(&z).Error; err == nil {
			query = query.Joins("JOIN zone_channels ON zone_channels.channel_id = channels.id").
				Where("zone_channels.zone_id = ?", z.ID).
				Order("sort_order ASC")
		}
	}
	query.Find(&channels)

	useFirstName := r.URL.Query().Get("use_first_name") == "true"

	// Handle Exports
	if format == "zip" {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=export.zip")
		zipWriter := zip.NewWriter(w)
		defer zipWriter.Close()

		// 1. Digital Contacts
		if radio == "dm32uv" {
			dcFile, err := zipWriter.Create("digital_contacts.csv")
			if err != nil {
				log.Printf("Error creating zip entry: %v", err)
				return
			}
			var bitContacts []models.DigitalContact
			// Note: We might want filtering logic here too, but for web export let's export all or apply simple limit
			database.DB.Limit(50000).Find(&bitContacts)
			exporter.ExportDM32UVDigitalContacts(bitContacts, dcFile)

			// 2. Talkgroups
			tgFile, err := zipWriter.Create("talkgroups.csv")
			if err != nil {
				log.Printf("Error creating zip entry: %v", err)
				return
			}
			var talkgroups []models.Contact
			database.DB.Where("type = ?", models.ContactTypeGroup).Find(&talkgroups)
			exporter.ExportDM32UVTalkgroups(talkgroups, tgFile)

			// 3. Channels
			chanFile, err := zipWriter.Create("channels.csv")
			if err != nil {
				log.Printf("Error creating zip entry: %v", err)
				return
			}
			exporter.ExportDM32UVChannels(channels, chanFile)

			// 4. Zones
			zoneFile, err := zipWriter.Create("zones.csv")
			if err != nil {
				log.Printf("Error creating zip entry: %v", err)
				return
			}
			var zones []models.Zone
			database.DB.Preload("Channels").Find(&zones)
			exporter.ExportDM32UVZones(zones, zoneFile)
		}
		return
	}

	tempFile, err := os.CreateTemp("", "export-*.csv")
	if err != nil {
		http.Error(w, "Error creating temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	// Open temp file for writing
	f, err := os.OpenFile(tempFile.Name(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		http.Error(w, "Error opening temp file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if format == "chirp" {
		exporter.ExportChirpCSV(channels, f)
	} else {
		exporter.ExportDB25D(channels, f, useFirstName)
	}

	http.ServeFile(w, r, tempFile.Name())
}

func handleContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		query := database.DB.Model(&models.Contact{})

		// Filtering
		// source := r.URL.Query().Get("source")
		// if source != "" {
		// 	query = query.Where("source = ?", source)
		// }

		// Search
		search := r.URL.Query().Get("search")
		if search != "" {
			searchLower := "%" + strings.ToLower(search) + "%"
			searchExact := "%" + search + "%"
			// Only Name and DMRID remain in Contact
			query = query.Where("LOWER(name) LIKE ? OR CAST(dmr_id AS TEXT) LIKE ?", searchLower, searchExact)
		}

		// Sorting
		sortField := r.URL.Query().Get("sort")
		order := r.URL.Query().Get("order")
		if order != "desc" {
			order = "asc"
		}

		switch sortField {
		case "dmr_id":
			query = query.Order("dmr_id " + order)
		case "name":
			query = query.Order("name " + order)
		default:
			query = query.Order("name ASC")
		}

		// Pagination
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 {
			limit = 100 // Default limit
		}
		offset := (page - 1) * limit

		var total int64
		query.Count(&total)

		var contacts []models.Contact
		result := query.Limit(limit).Offset(offset).Find(&contacts)
		if result.Error != nil {
			http.Error(w, result.Error.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": contacts,
			"meta": map[string]interface{}{
				"total": total,
				"page":  page,
				"limit": limit,
			},
		})
	} else if r.Method == "POST" {
		var c models.Contact
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := c.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var result *gorm.DB
		if c.ID == 0 {
			result = database.DB.Create(&c)
		} else {
			result = database.DB.Save(&c)
		}
		if result.Error != nil {
			http.Error(w, result.Error.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(c)
	} else if r.Method == "DELETE" {
		id := r.URL.Query().Get("id")
		if id != "" {
			// Check if used
			var count int64
			database.DB.Model(&models.Channel{}).Where("contact_id = ?", id).Count(&count)
			if count > 0 {
				http.Error(w, "Cannot delete contact that is in use by a channel", http.StatusConflict)
				return
			}

			database.DB.Delete(&models.Contact{}, id)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func resolveContacts(channels []models.Channel) {
	for i := range channels {
		if channels[i].TxContact != "" {
			var contact models.Contact
			// Find by Name
			if err := database.DB.Where("name = ?", channels[i].TxContact).First(&contact).Error; err == nil {
				channels[i].ContactID = &contact.ID
			}
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	hub.register <- conn

	// Send current status immediately
	broadcastProgress()

	// Keep connection alive/handle control messages if needed
	// For now, just block until close
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			hub.unregister <- conn
			break
		}
	}
}

func handleZones(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// List or Get One
		id := r.URL.Query().Get("id")
		if id != "" {
			var zone models.Zone
			// Preload Channels in Order
			if err := database.DB.Preload("Channels", func(db *gorm.DB) *gorm.DB {
				return db.Joins("JOIN zone_channels ON zone_channels.channel_id = channels.id").
					Order("zone_channels.sort_order ASC")
			}).First(&zone, id).Error; err != nil {
				http.Error(w, "Zone not found", http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(zone)
		} else {
			var zones []models.Zone
			database.DB.Preload("Channels", func(db *gorm.DB) *gorm.DB {
				return db.Joins("JOIN zone_channels ON zone_channels.channel_id = channels.id").
					Order("zone_channels.sort_order ASC")
			}).Find(&zones)
			json.NewEncoder(w).Encode(zones)
		}
	} else if r.Method == "POST" {
		var z models.Zone
		if err := json.NewDecoder(r.Body).Decode(&z); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if z.ID == 0 {
			database.DB.Create(&z)
		} else {
			database.DB.Save(&z)
		}
		json.NewEncoder(w).Encode(z)
	} else if r.Method == "DELETE" {
		id := r.URL.Query().Get("id")
		if id != "" {
			database.DB.Delete(&models.Zone{}, id)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func handleZoneAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Zone ID required", http.StatusBadRequest)
		return
	}

	var channelIDs []uint
	if err := json.NewDecoder(r.Body).Decode(&channelIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Transaction to replace associations with order
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var zone models.Zone
		if err := tx.First(&zone, id).Error; err != nil {
			return err
		}

		// Clear existing
		tx.Model(&zone).Association("Channels").Clear()

		// Add with Order
		// We insert manually into zone_channels to set sort_order
		for i, chID := range channelIDs {
			if err := tx.Create(&models.ZoneChannel{
				ZoneID:    zone.ID,
				ChannelID: chID,
				SortOrder: i, // 0-indexed order
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Error assigning channels: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
