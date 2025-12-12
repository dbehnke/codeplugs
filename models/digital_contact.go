package models

import (
	"gorm.io/gorm"
)

// DigitalContact represents a global DMR directory entry (e.g. from RadioID.net)
type DigitalContact struct {
	gorm.Model
	DMRID    int `gorm:"uniqueIndex"`
	Callsign string
	Name     string
	Country  string
	City     string
	State    string
	Remarks  string
}
