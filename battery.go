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

import "strings"

const (
	BatteryKey         = "battery"
	limeBatteryThreshV = 10
)

var LiIon = map[float32]float32{3.4: 0, 3.46: 5, 3.51: 10, 3.56: 15, 3.58: 20, 3.61: 25, 3.62: 30, 3.64: 35, 3.67: 40, 3.71: 45, 3.76: 50, 3.81: 55, 3.86: 60, 3.9: 65, 3.93: 70, 3.97: 75, 4.0: 80, 4.04: 85, 4.07: 90, 4.11: 95, 4.17: 100}
var Lime = map[float32]float32{30: 0, 30.1: 1, 30.2: 2, 30.4: 3, 30.5: 4, 30.6: 5, 30.7: 6, 30.8: 7, 31.0: 8, 31.1: 9, 31.2: 10, 31.3: 11, 31.4: 12, 31.6: 13, 31.7: 14, 31.8: 15, 31.9: 16, 32.0: 17, 32.2: 18, 32.3: 19, 32.4: 20, 32.5: 21, 32.6: 22, 32.8: 23, 32.9: 24, 33.0: 25, 33.1: 26, 33.2: 27, 33.4: 28, 33.5: 29, 33.6: 30, 33.7: 31, 33.8: 32, 34.0: 33, 34.1: 34, 34.2: 35, 34.3: 36, 34.4: 37, 34.6: 38, 34.7: 39, 34.8: 40, 34.9: 41, 35.0: 42, 35.2: 43, 35.3: 44, 35.4: 45, 35.5: 46, 35.6: 47, 35.8: 48, 35.9: 49, 36.0: 50, 36.1: 51, 36.2: 52, 36.4: 53, 36.5: 54, 36.6: 55, 36.7: 56, 36.8: 57, 37.0: 58, 37.1: 59, 37.2: 60, 37.3: 61, 37.4: 62, 37.6: 63, 37.7: 64, 37.8: 65, 37.9: 66, 38.0: 67, 38.2: 68, 38.3: 69, 38.4: 70, 38.5: 71, 38.6: 72, 38.8: 73, 38.9: 74, 39.0: 75, 39.1: 76, 39.2: 77, 39.4: 78, 39.5: 79, 39.6: 80, 39.7: 81, 39.8: 82, 40.0: 83, 40.1: 84, 40.2: 85, 40.3: 86, 40.4: 87, 40.6: 88, 40.7: 89, 40.8: 90, 40.9: 91, 41.0: 92, 41.2: 93, 41.3: 94, 41.4: 95, 41.5: 96, 41.6: 97, 41.8: 98, 41.9: 99, 42.0: 100}

func init() {
	allSections[BatteryKey] = section{
		key:         BatteryKey,
		mapToStruct: batteryMapToStruct,
		validate:    noValidateFunc,
	}
}

type Battery struct {
	EnableVoltageReadings    bool                `mapstructure:"enable-voltage-readings"`
	BatteryType              string              `mapstructure:"battery-type"`
	BatteryVoltageThresholds map[float32]float32 `mapstructure:"battery-voltage-thresholds"`
}

// https://imgur.com/IoUKfQs
func DefaultBattery() Battery {
	return Battery{
		EnableVoltageReadings: true,
	}
}

// GetBatteryVoltageThresholds gets battery type and voltage thresholds
// if no battery type is specific LiIon or Lime will be used based of batVolt reading
// if a battery type other than li-ion or lime is specified it will use the battery-voltage-thresholds map
// this should be  map in assending order of {voltage : percentage, voltage_2 : percentage_2....} of battery
func (batteryConfig *Battery) GetBatteryVoltageThresholds(batVolt float32) (string, map[float32]float32) {
	batType := strings.ToLower(strings.Trim(batteryConfig.BatteryType, ""))
	if batType == "li-ion" {
		return batType, LiIon
	} else if batType == "lime" {
		return batType, Lime
	} else if batType != "" {
		return batType, batteryConfig.BatteryVoltageThresholds
	}
	if batVolt <= limeBatteryThreshV {
		return "li-ion", LiIon
	}
	return "lime", Lime
}

func batteryMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s Battery
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}
