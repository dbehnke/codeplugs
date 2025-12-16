package models

import "gorm.io/gorm"

// ContactList represents a named collection of allowed DMR IDs
type ContactList struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex"`
	Description string
	Entries     []ContactListEntry `gorm:"constraint:OnDelete:CASCADE;"` // Cascade delete entries when list is deleted
}

// ContactListEntry represents a single DMR ID within a ContactList
type ContactListEntry struct {
	gorm.Model
	ContactListID uint `gorm:"index"`
	DMRID         int  `gorm:"index"`
}
