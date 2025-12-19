package importer

import (
	"os"
	"testing"

	"codeplugs/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func setupDM32UVTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file:memdb_dm32uv_imp?mode=memory&cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	// Migrate the schema
	err = db.AutoMigrate(&models.Channel{}, &models.Contact{}, &models.DigitalContact{}, &models.Zone{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	return db
}

func TestImportDM32UVChannels(t *testing.T) {
	db := setupDM32UVTestDB(t)

	// Create a temp CSV file

	tmpfile, err := os.CreateTemp("", "dm32uv_channels_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	content := `No.,Channel Name,Channel Type,RX Frequency[MHz],TX Frequency[MHz],Power,Band Width,Scan List,TX Admit,Emergency System,Squelch Level,APRS Report Type,Forbid TX,APRS Receive,Forbid Talkaround,Auto Scan,Lone Work,Emergency Indicator,Emergency ACK,Analog APRS PTT Mode,Digital APRS PTT Mode,TX Contact,RX Group List,Color Code,Time Slot,Encryption,Encryption ID,APRS Report Channel,Direct Dual Mode,Private Confirm,Short Data Confirm,DMR ID,CTC/DCS Decode,CTC/DCS Encode,Scramble,RX Squelch Mode,Signaling Type,PTT ID,VOX Function,PTT ID Display
1,TestCh1,Digital,440.00000,445.00000,High,12.5KHz,None,Always,None,3,Off,0,0,0,0,0,0,0,0,0,TestContact,None,1,Slot 1,0,None,1,0,0,0,MyID,None,None,None,Carrier/CTC,None,OFF,0,0`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	// Re-open for reading since Write closes or we can just keep open but easier to reopen or seek
	tmpfile.Seek(0, 0)

	if err := ImportDM32UVChannels(db, tmpfile); err != nil {
		t.Errorf("ImportDM32UVChannels failed: %v", err)
	}

	var count int64
	db.Model(&models.Channel{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 channel, got %d", count)
	}

	var ch models.Channel
	db.First(&ch)
	if ch.Name != "TestCh1" {
		t.Errorf("Expected name TestCh1, got %s", ch.Name)
	}
	if ch.RxFrequency != 440.0 {
		t.Errorf("Expected RxFreq 440.0, got %f", ch.RxFrequency)
	}
	if ch.Type != models.ChannelTypeDigitalDMR {
		t.Errorf("Expected Digital, got %s", ch.Type)
	}
}

func TestImportDM32UVTalkgroups(t *testing.T) {
	db := setupDM32UVTestDB(t)

	content := `No.,Name,ID,Type
1,TG1,12345,Group Call`

	tmpfile, err := os.CreateTemp("", "dm32uv_tg_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	// Re-open
	f, _ := os.Open(tmpfile.Name())
	defer f.Close()

	if err := ImportDM32UVTalkgroups(db, f); err != nil {
		t.Errorf("ImportDM32UVTalkgroups failed: %v", err)
	}

	var contact models.Contact
	if err := db.Where("dmr_id = ?", 12345).First(&contact).Error; err != nil {
		t.Errorf("Contact not found: %v", err)
	}
	if contact.Name != "TG1" {
		t.Errorf("Expected TG1, got %s", contact.Name)
	}
	if contact.Type != models.ContactTypeGroup {
		t.Errorf("Expected Group, got %s", contact.Type)
	}
}

func TestImportDM32UVZones(t *testing.T) {
	db := setupDM32UVTestDB(t)

	// Create dummy channels first
	db.Create(&models.Channel{Name: "Ch1"})
	db.Create(&models.Channel{Name: "Ch2"})

	content := `No.,Zone Name,Channel Members
1,ZoneA,Ch1|Ch2`

	tmpfile, err := os.CreateTemp("", "dm32uv_zones_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	f, _ := os.Open(tmpfile.Name())
	defer f.Close()

	if err := ImportDM32UVZones(db, f); err != nil {
		t.Errorf("ImportDM32UVZones failed: %v", err)
	}

	var zone models.Zone
	if err := db.Preload("Channels").First(&zone).Error; err != nil {
		t.Errorf("Zone not found: %v", err)
	}
	if zone.Name != "ZoneA" {
		t.Errorf("Expected ZoneA, got %s", zone.Name)
	}
	if len(zone.Channels) != 2 {
		t.Errorf("Expected 2 channels, got %d", len(zone.Channels))
	}
}

func TestImportDM32UVDigitalContacts(t *testing.T) {
	db := setupDM32UVTestDB(t)

	content := `No.,ID,Repeater,Name,City,Province,Country,Remark,Type,Alert Call
1,99999,Callsign,Name,City,Prov,Country,Rem,Private Call,0`

	tmpfile, err := os.CreateTemp("", "dm32uv_dc_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	f, _ := os.Open(tmpfile.Name())
	defer f.Close()

	if err := ImportDM32UVDigitalContacts(db, f); err != nil {
		t.Errorf("ImportDM32UVDigitalContacts failed: %v", err)
	}

	var dc models.DigitalContact
	if err := db.First(&dc).Error; err != nil {
		t.Errorf("DigitalContact not found: %v", err)
	}
	if dc.DMRID != 99999 {
		t.Errorf("Expected 99999, got %d", dc.DMRID)
	}
}
