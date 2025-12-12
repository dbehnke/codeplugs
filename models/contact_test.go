package models

import (
	"testing"
)

func TestContactValidation(t *testing.T) {
	t.Run("Valid Contact Types", func(t *testing.T) {
		validTypes := []ContactType{ContactTypeGroup, ContactTypePrivate, ContactTypeAllCall}
		for _, vt := range validTypes {
			c := Contact{Type: vt, Name: "Test", DMRID: 1}
			if err := c.Validate(); err != nil {
				t.Errorf("Expected %s to be valid, got: %v", vt, err)
			}
		}
	})

	t.Run("Invalid Contact", func(t *testing.T) {
		c := Contact{Type: "Invalid", Name: "Test", DMRID: 1}
		if err := c.Validate(); err == nil {
			t.Error("Expected error for invalid contact type")
		}
	})

	t.Run("Contact ID Requirement", func(t *testing.T) {
		c := Contact{Type: ContactTypeGroup, Name: "Test", DMRID: 0}
		if err := c.Validate(); err == nil {
			t.Error("Expected error for missing DMRID (0)")
		}
	})

}
