package models

import "gorm.io/gorm"

type RoamingChannel struct {
	gorm.Model
	Name        string  `json:"name"`
	RxFrequency float64 `json:"rx_frequency"`
	TxFrequency float64 `json:"tx_frequency"`
	ColorCode   int     `json:"color_code"`
	TimeSlot    int     `json:"time_slot"`
}

type RoamingZone struct {
	gorm.Model
	Name     string           `json:"name"`
	Channels []RoamingChannel `gorm:"many2many:roaming_zone_channels;" json:"channels"`
}

// Helper to find or create
func FindOrCreateRoamingZone(db *gorm.DB, name string) (*RoamingZone, error) {
	var zone RoamingZone
	err := db.Where("name = ?", name).FirstOrCreate(&zone, RoamingZone{Name: name}).Error
	return &zone, err
}
