package models

type ZoneChannel struct {
	ZoneID    uint `gorm:"primaryKey"`
	ChannelID uint `gorm:"primaryKey"`
	SortOrder int
}
