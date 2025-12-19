package services

import (
	"codeplugs/models"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

// FixBandwidths updates channel bandwidths to defaults (12.5 for Digital/Mixed, 25 for Analog)
func FixBandwidths(db *gorm.DB) (int, error) {
	fmt.Println("Fixing channel bandwidths...")
	var channels []models.Channel
	if err := db.Find(&channels).Error; err != nil {
		return 0, err
	}

	count := 0
	for _, ch := range channels {
		updated := false
		if ch.Type == models.ChannelTypeAnalog && ch.Bandwidth != "25" {
			ch.Bandwidth = "25"
			updated = true
		} else if (ch.IsDigital() || ch.Type == models.ChannelTypeMixed) && ch.Bandwidth != "12.5" {
			ch.Bandwidth = "12.5"
			updated = true
		}

		if updated {
			if err := db.Save(&ch).Error; err != nil {
				log.Printf("Failed to update channel %s: %v", ch.Name, err)
			} else {
				count++
			}
		}
	}
	return count, nil
}

// ResolveContacts links channels to existing contacts or creates new ones based on TxContact name
func ResolveContacts(db *gorm.DB, channels []models.Channel) {
	// Cache existing contacts
	contactMap := make(map[string]int)
	var contacts []models.Contact
	db.Find(&contacts)
	for _, c := range contacts {
		contactMap[strings.ToUpper(strings.TrimSpace(c.Name))] = int(c.ID)
	}

	for i := range channels {
		// Try to match TxContact string to a Contact
		if channels[i].TxContact != "" {
			nameUpper := strings.ToUpper(strings.TrimSpace(channels[i].TxContact))
			if id, ok := contactMap[nameUpper]; ok {
				uid := uint(id)
				channels[i].ContactID = &uid
			} else {
				// Auto-create?
				// We need a unique ID to satisfy the (dmr_id, type) constraint.
				// Since we don't know the ID, we'll assign a temporary negative ID.
				var minID int
				db.Model(&models.Contact{}).Select("MIN(dmr_id)").Scan(&minID)
				if minID > 0 {
					minID = 0
				}
				newID := minID - 1

				newContact := models.Contact{
					Name:  channels[i].TxContact,
					Type:  models.ContactTypeGroup, // Default to Group
					DMRID: newID,
				}
				if result := db.Create(&newContact); result.Error == nil {
					uid := newContact.ID
					channels[i].ContactID = &uid
					contactMap[nameUpper] = int(newContact.ID)
				} else {
					log.Printf("Failed to auto-create contact %s: %v", channels[i].TxContact, result.Error)
				}
			}
		}
	}
}
