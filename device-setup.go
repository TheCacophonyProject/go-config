// go-config - Library for reading cacophony config files.
// Copyright (C) 2018, The Cacophony Project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package config

const DeviceSetupKey = "device-setup"

func init() {
	allSections[DeviceSetupKey] = section{
		key:         DeviceSetupKey,
		mapToStruct: deviceSetupMapToStruct,
		validate:    noValidateFunc,
		defaultValue: func() interface{} {
			return DefaultDeviceSetup()
		},
		pointerValue: func() interface{} {
			return &DeviceSetup{}
		},
	}
}

type DeviceSetup struct {
	IR bool `mapstructure:"ir"`

	// S or L ( for small trap or large trap)
	TrapSize string `mapstructure:"trap-size"`
}

func DefaultDeviceSetup() DeviceSetup {
	return DeviceSetup{
		IR:       false,
		TrapSize: "s",
	}
}

func deviceSetupMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s DeviceSetup
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}
