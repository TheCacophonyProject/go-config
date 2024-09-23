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

const GPIOKey = "gpio"

func init() {
	allSections[GPIOKey] = section{
		key:         GPIOKey,
		mapToStruct: gpioMapToStruct,
		validate:    noValidateFunc,
		defaultValue: func() interface{} {
			return DefaultGPIO()
		},
		pointerValue: func() interface{} {
			return &GPIO{}
		},
	}
}

type GPIO struct {
	ThermalCameraPower string `mapstructure:"thermal-camera-power"`
	ModemPower         string `mapstructure:"modem-power"`
	UartTx             string `mapstructure:"uart-tx"`
}

func DefaultGPIO() GPIO {
	return GPIO{
		ThermalCameraPower: "GPIO23",
		ModemPower:         "GPIO22",
		UartTx:             "GPIO14",
	}
}

func gpioMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s GPIO
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}
