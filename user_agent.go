package hindsight

import (
	ua "github.com/mileusna/useragent"
)

type Device string

const (
	DeviceBot     Device = "bot"
	DeviceMobile  Device = "mobile"
	DeviceTablet  Device = "tablet"
	DeviceDesktop Device = "desktop"
	DeviceUnknown Device = "unknown"
)

type UAInfo struct {
	Browser NameAndVersion
	OS      NameAndVersion
	Device  Device
}

func DecodeUserAgent(uaString string) *UAInfo {
	info := ua.Parse(uaString)
	return &UAInfo{
		Browser: NameAndVersion{
			Name:    info.Name,
			Version: info.Version,
		},
		OS: NameAndVersion{
			Name:    info.OS,
			Version: info.OSVersion,
		},
		Device: getDevice(&info),
	}
}

func getDevice(info *ua.UserAgent) Device {
	switch {
	case info.Bot:
		return DeviceBot

	case info.Tablet:
		return DeviceTablet
	case info.Mobile:
		return DeviceMobile
	case info.Desktop:
		return DeviceDesktop
	default:
		return DeviceUnknown
	}
}
