package models

import "gorm.io/gorm"

type Zone struct {
	gorm.Model
	Name         string
	Channels     []Channel     `gorm:"many2many:zone_channels;"`
	ZoneChannels []ZoneChannel `gorm:"foreignKey:ZoneID"`
}

func FindOrCreateZone(db *gorm.DB, name string) (*Zone, error) {
	var zone Zone
	err := db.Where("name = ?", name).FirstOrCreate(&zone, Zone{Name: name}).Error
	return &zone, err
}
