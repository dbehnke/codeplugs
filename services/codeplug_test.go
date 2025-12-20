package services_test

import (
	"codeplugs/models"
	"codeplugs/services"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func setupServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	db.AutoMigrate(&models.Channel{}, &models.Contact{})
	return db
}

func TestFixBandwidths(t *testing.T) {
	db := setupServiceTestDB(t)

	// Seed Incorrect Data
	c1 := models.Channel{Name: "AnalogBad", Type: models.ChannelTypeAnalog, Bandwidth: "12.5"}
	c2 := models.Channel{Name: "DigitalBad", Type: models.ChannelTypeDigitalDMR, Bandwidth: "25"}
	c3 := models.Channel{Name: "AnalogGood", Type: models.ChannelTypeAnalog, Bandwidth: "25"}
	db.Create(&c1)
	db.Create(&c2)
	db.Create(&c3)

	count, err := services.FixBandwidths(db)
	if err != nil {
		t.Fatalf("FixBandwidths failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 fixes, got %d", count)
	}

	// Verify
	var checkC1, checkC2 models.Channel
	db.First(&checkC1, "name = ?", "AnalogBad")
	if checkC1.Bandwidth != "25" {
		t.Errorf("AnalogBad not fixed, got %s", checkC1.Bandwidth)
	}

	db.First(&checkC2, "name = ?", "DigitalBad")
	if checkC2.Bandwidth != "12.5" {
		t.Errorf("DigitalBad not fixed, got %s", checkC2.Bandwidth)
	}
}

func TestResolveContacts_AutoCreate(t *testing.T) {
	db := setupServiceTestDB(t)

	// Seed Channels with TxContact names but no IDs
	ch1 := models.Channel{Name: "Ch1", TxContact: "ExistingTG"}
	ch2 := models.Channel{Name: "Ch2", TxContact: "NewTG"}
	db.Create(&ch1)
	db.Create(&ch2)

	// Seed Existing Contact
	existing := models.Contact{Name: "ExistingTG", DMRID: 100, Type: models.ContactTypeGroup}
	db.Create(&existing)

	// Run Resolution
	var channels []models.Channel
	db.Find(&channels)

	// Note: ResolveContacts modifies the slice *in place* but doesn't necessarily save to DB?
	// Looking at code: it sets pointer `channels[i].ContactID`. It relies on caller to save?
	// Let's check code: "resolveContacts... db.Create(&newContact)..." but for the channel linking "channels[i].ContactID = &uid". It does NOT save the channel itself.
	// But it does save the NEW contact.
	services.ResolveContacts(db, channels)

	// Verify Existing Link
	// Find correct channel in slice
	var ch1Idx, ch2Idx int = -1, -1
	for i, c := range channels {
		if c.Name == "Ch1" {
			ch1Idx = i
		} else if c.Name == "Ch2" {
			ch2Idx = i
		}
	}

	if ch1Idx == -1 {
		t.Fatal("Ch1 not found in channels slice")
	}

	if channels[ch1Idx].ContactID == nil {
		t.Errorf("Ch1 ContactID is nil. TxContact: '%s'", channels[ch1Idx].TxContact)
	} else if *channels[ch1Idx].ContactID != existing.ID {
		t.Errorf("Ch1 linked to wrong ID %d, expected %d", *channels[ch1Idx].ContactID, existing.ID)
	}

	// Verify New Contact Creation
	var newTG models.Contact
	if err := db.Where("name = ?", "NewTG").First(&newTG).Error; err != nil {
		t.Fatal("NewTG contact not created")
	}
	if newTG.DMRID >= 0 {
		t.Errorf("NewTG should have negative ID, got %d", newTG.DMRID)
	}

	// Verify New Link
	if ch2Idx == -1 {
		t.Fatal("Ch2 not found in channels slice")
	}
	if channels[ch2Idx].ContactID == nil {
		t.Errorf("Ch2 ContactID is nil. TxContact: '%s'", channels[ch2Idx].TxContact)
	} else if *channels[ch2Idx].ContactID != newTG.ID {
		t.Errorf("Ch2 linked to wrong ID %d, expected %d", *channels[ch2Idx].ContactID, newTG.ID)
	}
}
