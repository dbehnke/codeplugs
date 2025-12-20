package models

import "gorm.io/gorm"

type Zone struct {
	gorm.Model
	Name         string        `json:"name"`
	Channels     []Channel     `gorm:"many2many:zone_channels;" json:"channels"`
	ZoneChannels []ZoneChannel `gorm:"foreignKey:ZoneID" json:"-"`
}

func FindOrCreateZone(db *gorm.DB, name string) (*Zone, error) {
	var zone Zone
	err := db.Where("name = ?", name).FirstOrCreate(&zone, Zone{Name: name}).Error
	return &zone, err
}
