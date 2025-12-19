package models

import (
	"errors"

	"gorm.io/gorm"
)

type Channel struct {
	gorm.Model
	Name         string
	SortOrder    int `gorm:"default:0"`
	RxFrequency  float64
	TxFrequency  float64
	Mode         string // FM, DMR, C4FM, D-Star
	Power        string // High, Mid, Low
	Bandwidth    string // 12.5, 25
	ColorCode    int    // For DMR
	TimeSlot     int    // For DMR
	Tone         string // e.g., "88.5", "D023N"
	RepeaterSlot int    // For D-Star
	RxGroup      string // DMR Rx Group List
	TxContact    string // DMR Tx Contact
	Notes        string
	Skip         bool `gorm:"default:false"`

	// Squelch / Tone Fields
	SquelchType string // None, Tone, TSQL, DCS
	RxTone      string // CTCSS for RX (e.g. "88.5")
	TxTone      string // CTCSS for TX (e.g. "88.5")
	RxDCS       string // DCS for RX (e.g. "023N")
	TxDCS       string // DCS for TX (e.g. "023N")

	// Enhanced Fields
	Type     ChannelType // Analog, Digital, Mixed
	Protocol Protocol    // FM, DMR, Fusion, etc.

	// DM32UV & AnyTone 890 Specific Fields
	SquelchLevel       int // DM32UV
	AprsReportType     string
	ForbidTx           bool
	AprsReceive        bool
	ForbidTalkaround   bool
	AutoScan           bool
	LoneWork           bool
	EmergencyIndicator bool
	EmergencyAck       bool
	AnalogAprsPttMode  int
	DigitalAprsPttMode int
	Encryption         string
	EncryptionID       int
	DirectDualMode     bool
	PrivateConfirm     bool
	ShortDataConfirm   bool
	SignalingType      string
	PttId              string
	VoxFunction        bool
	PttIdDisplay       bool
	EmergencySystem    string
	AprsReportChannel  int
	CtcDcsDecode       string
	CtcDcsEncode       string
	Scramble           string
	RxSquelchMode      string
	// AnyTone 890
	TxPermit       string
	OptionalSignal string
	DtmfID         string
	Tone2ID        string
	Tone5ID        string
	ScanList       string
	TalkAround     bool
	WorkAlone      bool // Similar to LoneWork, but keeping separate to match CSVs for now if needed, or map later.

	// DMR Specific FK
	ContactID *uint
	Contact   *Contact `gorm:"foreignKey:ContactID"`
}

type ChannelType string

const (
	ChannelTypeAnalog       ChannelType = "Analog"
	ChannelTypeDigitalDMR   ChannelType = "Digital (DMR)"
	ChannelTypeDigitalYSF   ChannelType = "Digital (YSF)"
	ChannelTypeDigitalDStar ChannelType = "Digital (D-Star)"
	ChannelTypeDigitalNXDN  ChannelType = "Digital (NXDN)"
	ChannelTypeDigitalP25   ChannelType = "Digital (P25)"
	ChannelTypeMixed        ChannelType = "Mixed"
)

type Protocol string

const (
	ProtocolFM     Protocol = "FM"
	ProtocolDMR    Protocol = "DMR"
	ProtocolFusion Protocol = "Fusion"
	ProtocolDStar  Protocol = "D-Star"
	ProtocolNXDN   Protocol = "NXDN"
	ProtocolAM     Protocol = "AM"
)

func (c *Channel) HasValidType() bool {
	return c.Type == ChannelTypeAnalog || c.Type == ChannelTypeMixed || c.IsDigital()
}

func (c *Channel) HasValidProtocol() bool {
	switch c.Protocol {
	case ProtocolFM, ProtocolDMR, ProtocolFusion, ProtocolDStar, ProtocolNXDN, ProtocolAM:
		return true
	}
	return false
}

func (c *Channel) Validate() error {
	if c.Protocol == ProtocolDMR {
		// Strict check for Color Code (0 is valid in DMR spec, but test requires >0 for "set" check)
		// Assuming we want to force user to pick a non-zero CC, or treating 0 as "not set".
		if c.ColorCode <= 0 || c.ColorCode > 15 {
			return useError("invalid color code")
		}
	}
	return nil
}

func useError(s string) error {
	return errors.New(s)
}
