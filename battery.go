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

import (
	"fmt"
)

const (
	BatteryKey         = "battery"
	limeBatteryThreshV = 10
	noBatteryThreshV   = 0.2
)

func init() {
	allSections[BatteryKey] = section{
		key:         BatteryKey,
		mapToStruct: batteryMapToStruct,
		validate:    batteryValidateFunc,
		defaultValue: func() interface{} {
			return DefaultBattery()
		},
		pointerValue: func() interface{} {
			return &Battery{}
		},
	}
}

func batteryValidateFunc(batteryInterface interface{}) error {
	batteryStruct, err := ConvertToStruct[*Battery](batteryInterface)
	if err != nil {
		return err
	}
	if batteryStruct.PresetBatteryType != "" && batteryStruct.CustomBatteryType != nil {
		return fmt.Errorf("cannot specify both battery-type-name and custom-battery-type")
	}

	if batteryStruct.CustomBatteryType != nil {
		if len(batteryStruct.CustomBatteryType.Voltages) != len(batteryStruct.CustomBatteryType.Percent) {
			return fmt.Errorf("custom-battery-type must have the same number of voltages and percentages")
		}
	}
	return nil
}

// Battery struct. Only CustomBatteryYype or PresetBatteryType should be set. Not both.
type Battery struct {
	EnableVoltageReadings   bool         `mapstructure:"enable-voltage-readings"`
	CustomBatteryType       *BatteryType `mapstructure:"custom-battery-type"`       // When wanting to specify a custom battery type
	PresetBatteryType       string       `mapstructure:"preset-battery-type"`       // When wanting to use one from the default list.
	MinimumVoltageDetection float32      `mapstructure:"minimum-voltage-detection"` // Voltages below this will be considered 0V
}

func DefaultBattery() Battery {
	return Battery{
		EnableVoltageReadings:   true,
		MinimumVoltageDetection: 1.0,
	}
}

type BatteryType struct {
	Name       string
	MinVoltage float32 // This is the minimum voltage that this battery type will be, this is used to guess what battery type is in use if not specified.
	MaxVoltage float32 // This is the maximum voltage that this battery type will be (including when charging), this is used to guess what battery type is in use if not specified.
	Voltages   []float32
	Percent    []float32
}

// PresetBatteryTypes is a list of battery chemistries for the camera to choose from.
// Have the more preferred battery chemistries first (when auto detecting battery type
// from voltage multiple might be valid so it will choose the first one that is valid from hte list).
var PresetBatteryTypes = []BatteryType{
	LimeBattery,
	LiIonBattery,
	LeadAcid12V,
}

var LiIonBattery = BatteryType{
	Name:       "li-ion",
	MinVoltage: 2.9,
	MaxVoltage: 4.3,
	Voltages:   []float32{3.4, 3.46, 3.51, 3.56, 3.58, 3.61, 3.62, 3.64, 3.67, 3.71, 3.76, 3.81, 3.86, 3.90, 3.93, 3.97, 4.00, 4.04, 4.07, 4.11, 4.170},
	Percent:    []float32{0.0, 5.00, 10.0, 15.0, 20.0, 25.0, 30.0, 35.0, 40.0, 45.0, 50.0, 55.0, 60.0, 65.0, 70.0, 75.0, 80.0, 85.0, 90.0, 95.0, 100.0},
}

var LimeBattery = BatteryType{
	Name:       "lime",
	MinVoltage: 29,
	MaxVoltage: 42.5,
	Voltages:   []float32{30.0, 30.1, 30.2, 30.4, 30.5, 30.6, 30.7, 30.8, 31.0, 31.1, 31.2, 31.3, 31.4, 31.6, 31.7, 31.8, 31.9, 32.0, 32.2, 32.3, 32.4, 32.5, 32.6, 32.8, 32.9, 33.0, 33.1, 33.2, 33.4, 33.5, 33.6, 33.7, 33.8, 34.0, 34.1, 34.2, 34.3, 34.4, 34.6, 34.7, 34.8, 34.9, 35.0, 35.2, 35.3, 35.4, 35.5, 35.6, 35.8, 35.9, 36.0, 36.1, 36.2, 36.4, 36.5, 36.6, 36.7, 36.8, 37.0, 37.1, 37.2, 37.3, 37.4, 37.6, 37.7, 37.8, 37.9, 38.0, 38.2, 38.3, 38.4, 38.5, 38.6, 38.8, 38.9, 39.0, 39.1, 39.2, 39.4, 39.5, 39.6, 39.7, 39.8, 40.0, 40.1, 40.2, 40.3, 40.4, 40.6, 40.7, 40.8, 40.9, 41.0, 41.2, 41.3, 41.4, 41.5, 41.6, 41.8, 41.9, 42.0},
	Percent:    []float32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100},
}

var LeadAcid12V = BatteryType{
	Name:       "lead-acid-12v",
	MinVoltage: 9.0,
	MaxVoltage: 14.0,
	Voltages:   []float32{11.3, 11.5, 11.66, 11.81, 11.96, 12.1, 12.24, 12.37, 12.50, 12.62, 12.73},
	Percent:    []float32{0.00, 10.0, 20.00, 30.00, 40.00, 50.0, 60.00, 70.00, 80.00, 90.00, 100.0},
}

func batteryMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s Battery
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}
