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

import "time"

const CommsKey = "comms"

func init() {
	allSections[CommsKey] = section{
		key:         CommsKey,
		mapToStruct: commsMapToStruct,
		validate:    noValidateFunc,
		defaultValue: func() interface{} {
			return DefaultComms()
		},
		pointerValue: func() interface{} {
			return &Comms{}
		},
	}
}

type Comms struct {
	Enable               bool   `mapstructure:"enable"`
	TrapEnabledByDefault bool   `mapstructure:"trap-enabled-by-default"` // If no animals are seen should the trap be enabled or not.
	CommsOut             string `mapstructure:"comms-out"`               // "uart" or "high-low"
	Bluetooth            bool   `mapstructure:"bluetooth"`               // Bluetooth can only be enabled if UART is not in use.

	PowerOutput     string        `mapstructure:"power-output"`      // "on", "off", "comms-only"
	PowerUpDuration time.Duration `mapstructure:"power-up-duration"` // When PowerOutput is set to "comms-only" how long should it be powered up before sending data.

	TrapSpecies  map[string]int32 `mapstructure:"trap-species"`  // Species with set confidence to trap
	TrapDuration time.Duration    `mapstructure:"trap-duration"` // How long to keep a trap active for after seeing a trapped species

	ProtectSpecies  map[string]int32 `mapstructure:"protect-species"`  // Species with set confidence to protect
	ProtectDuration time.Duration    `mapstructure:"protect-duration"` // How long to keep a trap inactive for after seeing a protected species

}

func DefaultComms() Comms {
	return Comms{
		Enable:          false,
		TrapDuration:    time.Minute,
		ProtectDuration: time.Minute,
	}
}

func commsMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s Comms
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}
