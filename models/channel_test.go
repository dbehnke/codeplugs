package models

import (
	"testing"
)

func TestChannelValidation(t *testing.T) {
	// 1. Test Channel Types
	t.Run("Valid Channel Types", func(t *testing.T) {
		validTypes := []ChannelType{ChannelTypeAnalog, ChannelTypeDigital, ChannelTypeMixed}
		for _, vt := range validTypes {
			c := Channel{Type: vt}
			if !c.HasValidType() {
				t.Errorf("Expected %s to be valid", vt)
			}
		}
	})

	t.Run("Invalid Channel Type", func(t *testing.T) {
		c := Channel{Type: "InvalidType"}
		if c.HasValidType() {
			t.Errorf("Expected InvalidType to be invalid")
		}
	})

	// 2. Test Protocols
	t.Run("Valid Protocols", func(t *testing.T) {
		validProtocols := []Protocol{ProtocolFM, ProtocolDMR, ProtocolFusion, ProtocolDStar, ProtocolNXDN}
		for _, vp := range validProtocols {
			c := Channel{Protocol: vp}
			if !c.HasValidProtocol() {
				t.Errorf("Expected %s to be valid", vp)
			}
		}
	})

	// 3. Test DMR Requirements
	t.Run("DMR Validation", func(t *testing.T) {
		// Valid DMR
		c := Channel{
			Type:      ChannelTypeDigital,
			Protocol:  ProtocolDMR,
			ColorCode: 1,
			TimeSlot:  1,
		}
		if err := c.Validate(); err != nil {
			t.Errorf("Expected valid DMR channel, got error: %v", err)
		}

		// Invalid DMR (Missing Color Code)
		cInvalid := Channel{
			Type:      ChannelTypeDigital,
			Protocol:  ProtocolDMR,
			ColorCode: 0, // Invalid (usually 0-15, but 0 often means not set)
			TimeSlot:  1,
		}
		// Assuming 0 is treated as invalid for ColorCode in strict mode, or checks range
		// Let's implement Validate to check range 0-15
		if err := cInvalid.Validate(); err == nil {
			t.Error("Expected error for missing ColorCode on DMR channel")
		}
	})
}
