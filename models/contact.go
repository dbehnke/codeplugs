package models

import (
	"errors"

	"gorm.io/gorm"
)

type ContactType string

const (
	ContactTypeGroup   ContactType = "Group"
	ContactTypePrivate ContactType = "Private"
	ContactTypeAllCall ContactType = "AllCall"
)

type Contact struct {
	gorm.Model
	Name  string
	DMRID int         `gorm:"index:idx_dmr_id_type,unique"` // The actual Talkgroup ID or Private ID
	Type  ContactType `gorm:"index:idx_dmr_id_type,unique"` // Group, Private, AllCall
}

// Validate checks if the contact layout is valid
func (c *Contact) Validate() error {
	if c.Type != ContactTypeGroup && c.Type != ContactTypePrivate && c.Type != ContactTypeAllCall {
		return errors.New("invalid contact type")
	}
	if c.DMRID <= 0 {
		return errors.New("invalid DMR ID")
	}
	return nil
}
