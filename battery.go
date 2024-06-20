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
	noBatteryThreshV   = 0.2
)

var LiIonVoltage = []float32{3.4, 3.46, 3.51, 3.56, 3.58, 3.61, 3.62, 3.64, 3.67, 3.71, 3.76, 3.81, 3.86, 3.9, 3.93, 3.97, 4.0, 4.04, 4.07, 4.11, 4.17}
var LiIonPercent = []float32{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100}

var LimeVoltage = []float32{30.1, 30.2, 30.4, 30.5, 30.6, 30.7, 30.8, 31.0, 31.1, 31.2, 31.3, 31.4, 31.6, 31.7, 31.8, 31.9, 32.0, 32.2, 32.3, 32.4, 32.5, 32.6, 32.8, 32.9, 33.0, 33.1, 33.2, 33.4, 33.5, 33.6, 33.7, 33.8, 34.0, 34.1, 34.2, 34.3, 34.4, 34.6, 34.7, 34.8, 34.9, 35.0, 35.2, 35.3, 35.4, 35.5, 35.6, 35.8, 35.9, 36.0, 36.1, 36.2, 36.4, 36.5, 36.6, 36.7, 36.8, 37.0, 37.1, 37.2, 37.3, 37.4, 37.6, 37.7, 37.8, 37.9, 38.0, 38.2, 38.3, 38.4, 38.5, 38.6, 38.8, 38.9, 39.0, 39.1, 39.2, 39.4, 39.5, 39.6, 39.7, 39.8, 40.0, 40.1, 40.2, 40.3, 40.4, 40.6, 40.7, 40.8, 40.9, 41.0, 41.2, 41.3, 41.4, 41.5, 41.6, 41.8, 41.9, 42.0}
var LimePercent = []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100}

func init() {
	allSections[BatteryKey] = section{
		key:         BatteryKey,
		mapToStruct: batteryMapToStruct,
		validate:    noValidateFunc,
	}
}

type Battery struct {
	EnableVoltageReadings bool      `mapstructure:"enable-voltage-readings"`
	BatteryType           string    `mapstructure:"battery-type"`
	BatteryVoltage        []float32 `mapstructure:"battery-voltage"`
	BatteryPercent        []float32 `mapstructure:"battery-percent"`
	// map[float32]float32 `mapstructure:"battery-voltage-thresholds"`
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
func (batteryConfig *Battery) GetBatteryVoltageThresholds(batVolt float32) (string, []float32, []float32) {
	batType := strings.ToLower(strings.Trim(batteryConfig.BatteryType, ""))
	if batType == "li-ion" {
		return batType, LiIonVoltage, LiIonPercent
	} else if batType == "lime" {
		return batType, LimeVoltage, LimePercent
	} else if batType != "" {
		return batType, batteryConfig.BatteryVoltage, batteryConfig.BatteryPercent
	} else if batVolt <= noBatteryThreshV {
		return "mains", []float32{0, 0.2}, []float32{100, 100}
	} else if batVolt <= limeBatteryThreshV {
		return "li-ion", LiIonVoltage, LiIonPercent
	}
	return "lime", LimeVoltage, LimePercent
}

func batteryMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s Battery
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}
