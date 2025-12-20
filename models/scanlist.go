package models

import "gorm.io/gorm"

type ScanList struct {
	gorm.Model
	Name     string    `json:"name"`
	Channels []Channel `gorm:"many2many:scan_list_channels;" json:"channels"`
}

type ScanListChannel struct {
	ScanListID uint `gorm:"primaryKey"`
	ChannelID  uint `gorm:"primaryKey"`
}

func FindOrCreateScanList(db *gorm.DB, name string) (*ScanList, error) {
	var list ScanList
	err := db.Where("name = ?", name).FirstOrCreate(&list, ScanList{Name: name}).Error
	return &list, err
}
