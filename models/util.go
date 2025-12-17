package models

func (c *Channel) IsDigital() bool {
	switch c.Type {
	case ChannelTypeDigitalDMR, ChannelTypeDigitalYSF, ChannelTypeDigitalDStar, ChannelTypeDigitalNXDN, ChannelTypeDigitalP25:
		return true
	default:
		return false
	}
}
