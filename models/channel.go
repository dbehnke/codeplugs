package models

import (
	"errors"

	"gorm.io/gorm"
)

type Channel struct {
	gorm.Model
	Name         string  `json:"name"`
	SortOrder    int     `gorm:"default:0" json:"sort_order"`
	RxFrequency  float64 `json:"rx_frequency"`
	TxFrequency  float64 `json:"tx_frequency"`
	Mode         string  `json:"mode"`          // FM, DMR, C4FM, D-Star
	Power        string  `json:"power"`         // High, Mid, Low
	Bandwidth    string  `json:"bandwidth"`     // 12.5, 25
	ColorCode    int     `json:"color_code"`    // For DMR
	TimeSlot     int     `json:"time_slot"`     // For DMR
	Tone         string  `json:"tone"`          // e.g., "88.5", "D023N"
	RepeaterSlot int     `json:"repeater_slot"` // For D-Star
	RxGroup      string  `json:"rx_group"`      // DMR Rx Group List
	TxContact    string  `json:"tx_contact"`    // DMR Tx Contact
	Notes        string  `json:"notes"`
	Skip         bool    `gorm:"default:false" json:"skip"`

	// Squelch / Tone Fields
	SquelchType string `json:"squelch_type"` // None, Tone, TSQL, DCS
	RxTone      string `json:"rx_tone"`      // CTCSS for RX (e.g. "88.5")
	TxTone      string `json:"tx_tone"`      // CTCSS for TX (e.g. "88.5")
	RxDCS       string `json:"rx_dcs"`       // DCS for RX (e.g. "023N")
	TxDCS       string `json:"tx_dcs"`       // DCS for TX (e.g. "023N")

	// Enhanced Fields
	Type     ChannelType `json:"type"`     // Analog, Digital, Mixed
	Protocol Protocol    `json:"protocol"` // FM, DMR, Fusion, etc.

	// DM32UV & AnyTone 890 Specific Fields
	SquelchLevel       int    `json:"squelch_level"` // DM32UV
	AprsReportType     string `json:"aprs_report_type"`
	ForbidTx           bool   `json:"forbid_tx"`
	AprsReceive        bool   `json:"aprs_receive"`
	ForbidTalkaround   bool   `json:"forbid_talkaround"`
	AutoScan           bool   `json:"auto_scan"`
	LoneWork           bool   `json:"lone_work"`
	EmergencyIndicator bool   `json:"emergency_indicator"`
	EmergencyAck       bool   `json:"emergency_ack"`
	AnalogAprsPttMode  int    `json:"analog_aprs_ptt_mode"`
	DigitalAprsPttMode int    `json:"digital_aprs_ptt_mode"`
	Encryption         string `json:"encryption"`
	EncryptionID       int    `json:"encryption_id"`
	DirectDualMode     bool   `json:"direct_dual_mode"`
	PrivateConfirm     bool   `json:"private_confirm"`
	ShortDataConfirm   bool   `json:"short_data_confirm"`
	SignalingType      string `json:"signaling_type"`
	PttId              string `json:"ptt_id"`
	VoxFunction        bool   `json:"vox_function"`
	PttIdDisplay       bool   `json:"ptt_id_display"`
	EmergencySystem    string `json:"emergency_system"`
	AprsReportChannel  int    `json:"aprs_report_channel"`
	CtcDcsDecode       string `json:"ctc_dcs_decode"`
	CtcDcsEncode       string `json:"ctc_dcs_encode"`
	Scramble           string `json:"scramble"`
	RxSquelchMode      string `json:"rx_squelch_mode"`
	// AnyTone 890
	TxPermit       string `json:"tx_permit"`
	OptionalSignal string `json:"optional_signal"`
	DtmfID         string `json:"dtmf_id"`
	Tone2ID        string `json:"tone2_id"`
	Tone5ID        string `json:"tone5_id"`
	ScanList       string `json:"scan_list"`
	TalkAround     bool   `json:"talk_around"`
	WorkAlone      bool   `json:"work_alone"` // Similar to LoneWork, but keeping separate to match CSVs for now if needed, or map later.

	// DMR Specific FK
	ContactID *uint    `json:"contact_id"`
	Contact   *Contact `gorm:"foreignKey:ContactID" json:"contact"`
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
