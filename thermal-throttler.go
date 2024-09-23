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

const ThermalThrottlerKey = "thermal-throttler"

func init() {
	allSections[ThermalThrottlerKey] = section{
		key:         ThermalThrottlerKey,
		mapToStruct: thermalThrottlerMapToStruct,
		validate:    noValidateFunc,
		defaultValue: func() interface{} {
			return DefaultThermalThrottler()
		},
		pointerValue: func() interface{} {
			return &ThermalThrottler{}
		},
	}
}

type ThermalThrottler struct {
	Activate   bool
	BucketSize time.Duration `mapstructure:"bucket-size"`
	MinRefill  time.Duration `mapstructure:"min-refill"`
}

func DefaultThermalThrottler() ThermalThrottler {

	return ThermalThrottler{
		Activate:   true,
		BucketSize: 10 * time.Minute,
		MinRefill:  10 * time.Minute,
	}
}

func thermalThrottlerMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s ThermalThrottler
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}
