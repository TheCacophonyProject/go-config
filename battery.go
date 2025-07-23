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
	"slices"
	"strings"
)

// Battery configuration constants
const (
	BatteryKey = "battery"

	// Voltage rails
	RailHV   = "hv"
	RailLV   = "lv"
	RailAuto = "auto"

	ChemistryLeadAcid = "lead-acid"
	ChemistryLiFePO4  = "lifepo4"
	ChemistryLiIon    = "li-ion"
	ChemistryLiPo     = "lipo"
	ChemistryCustom   = "custom"
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

// Battery represents the main battery configuration
type Battery struct {
	EnableVoltageReadings       bool         `mapstructure:"enable-voltage-readings"`
	CustomBatteryType           *BatteryType `mapstructure:"custom-battery-type"`
	PresetBatteryType           string       `mapstructure:"preset-battery-type"`
	MinimumVoltageDetection     float32      `mapstructure:"minimum-voltage-detection"`
	EnableDepletionEstimate     bool         `mapstructure:"enable-depletion-estimate"`
	DepletionHistoryHours       int          `mapstructure:"depletion-history-hours"`
	DepletionWarningHours       float32      `mapstructure:"depletion-warning-hours"`
	PowerSavingDischargeRatio   float32      `mapstructure:"power-saving-discharge-ratio"`
}

// DefaultBattery returns default battery configuration
func DefaultBattery() Battery {
	return Battery{
		EnableVoltageReadings:     true,
		MinimumVoltageDetection:   1.0,
		EnableDepletionEstimate:   true,
		DepletionHistoryHours:     48,
		DepletionWarningHours:     12.0,
		PowerSavingDischargeRatio: 0.3,
	}
}

// BatteryType defines a battery's characteristics
type BatteryType struct {
	Name       string  `mapstructure:"name"`
	Chemistry  string  `mapstructure:"chemistry"`
	MinVoltage float32 `mapstructure:"min-voltage"`
	MaxVoltage float32 `mapstructure:"max-voltage"`

	// Discharge curve
	Voltages []float32 `mapstructure:"voltages"`
	Percent  []float32 `mapstructure:"percent"`
}

// NormalizeCurves ensures voltage curves are properly set with backward compatibility
func (bt *BatteryType) NormalizeCurves() {
	// Set chemistry if not specified
	if bt.Chemistry == "" {
		bt.Chemistry = ChemistryCustom
	}
}

// PresetBatteryTypes ordered by descending MinVoltage for auto-detection priority
var PresetBatteryTypes = []BatteryType{
	LimeBattery,
	LiIonBattery,
	LiFePO4_24V,
	LeadAcid24V,
	LiFePO4_12V,
	LeadAcid12V,
	LiFePO4_6V,
}

var LiIonBattery = BatteryType{
	Name:       "li-ion",
	Chemistry:  ChemistryLiIon,
	MinVoltage: 2.9,
	MaxVoltage: 4.3,
	Voltages:   []float32{3.4, 3.46, 3.51, 3.56, 3.58, 3.61, 3.62, 3.64, 3.67, 3.71, 3.76, 3.81, 3.86, 3.90, 3.93, 3.97, 4.00, 4.04, 4.07, 4.11, 4.170},
	Percent:    []float32{0.0, 5.00, 10.0, 15.0, 20.0, 25.0, 30.0, 35.0, 40.0, 45.0, 50.0, 55.0, 60.0, 65.0, 70.0, 75.0, 80.0, 85.0, 90.0, 95.0, 100.0},
}

var LimeBattery = BatteryType{
	Name:       "lime",
	Chemistry:  ChemistryLiIon,
	MinVoltage: 29.0,
	MaxVoltage: 42.5,
	Voltages: []float32{
		30.0, 31.2, 32.4, 33.6, 34.8, 36.0, 37.2, 38.4, 39.6, 40.8, 42.0,
	},
	Percent: []float32{
		0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100,
	},
}

var LiFePO4_24V = BatteryType{
	Name:       "lifepo4-24v",
	Chemistry:  ChemistryLiFePO4,
	MinVoltage: 20.0,
	MaxVoltage: 29.2,
	Voltages: []float32{
		20.0, 24.0, 25.6, 25.8, 26.0, 26.1, 26.1, 26.4, 26.6, 26.8, 27.2,
	},
	Percent: []float32{
		0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100,
	},
}

var LiFePO4_12V = BatteryType{
	Name:       "lifepo4-12v",
	Chemistry:  ChemistryLiFePO4,
	MinVoltage: 10.0,
	MaxVoltage: 14.6,
	Voltages: []float32{
		10.0, 12.0, 12.8, 12.9, 13.0, 13.0, 13.1, 13.2, 13.3, 13.4, 13.6,
	},
	Percent: []float32{
		0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100,
	},
}

var LeadAcid24V = BatteryType{
	Name:       "lead-acid-24v",
	Chemistry:  ChemistryLeadAcid,
	MinVoltage: 22.0,
	MaxVoltage: 25.4,
	Voltages: []float32{
		23.18, 23.27, 23.51, 23.74, 23.94, 24.14, 24.36, 24.58, 24.81, 25.05, 25.29,
	},
	Percent: []float32{
		0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100,
	},
}

var LeadAcid12V = BatteryType{
	Name:       "lead-acid-12v",
	Chemistry:  ChemistryLeadAcid,
	MinVoltage: 9.0,
	MaxVoltage: 14.0,
	Voltages: []float32{
		11.59, 11.63, 11.76, 11.87, 11.97, 12.07, 12.18, 12.29, 12.41, 12.53, 12.64,
	},
	Percent: []float32{
		0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100,
	},
}

var LiFePO4_6V = BatteryType{
	Name:       "lifepo4-6v",
	Chemistry:  ChemistryLiFePO4,
	MinVoltage: 5.0,
	MaxVoltage: 7.3,
	Voltages: []float32{
		5.0, 6.0, 6.3, 6.4, 6.45, 6.5, 6.55, 6.6, 6.7, 6.8, 7.2,
	},
	Percent: []float32{
		0, 5, 10, 20, 30, 50, 70, 80, 90, 95, 100,
	},
}

// GetBatteryType returns the configured battery type (preset or custom)
func (b *Battery) GetBatteryType() *BatteryType {
	if b.CustomBatteryType != nil {
		bt := *b.CustomBatteryType
		bt.NormalizeCurves()
		return &bt
	}

	if b.PresetBatteryType != "" {
		for _, preset := range PresetBatteryTypes {
			if strings.EqualFold(preset.Name, b.PresetBatteryType) {
				bt := preset
				bt.NormalizeCurves()
				return &bt
			}
		}
	}

	return nil // Auto-detect mode
}

// batteryValidateFunc validates battery configuration
func batteryValidateFunc(battery any) error {
	b, ok := battery.(Battery)
	if !ok {
		return fmt.Errorf("invalid battery configuration type")
	}

	// Validate custom battery type if provided
	if b.CustomBatteryType != nil {
		bt := b.CustomBatteryType

		if bt.Name == "" {
			return fmt.Errorf("custom battery type must have a name")
		}

		if bt.MinVoltage >= bt.MaxVoltage {
			return fmt.Errorf("min voltage must be less than max voltage")
		}

		if len(bt.Voltages) == 0 {
			return fmt.Errorf("battery type must have voltage curve")
		}

		if len(bt.Voltages) != len(bt.Percent) {
			return fmt.Errorf("voltage and percent arrays must have same length")
		}

		// Validate chemistry if specified
		validChemistries := []string{ChemistryLeadAcid, ChemistryLiFePO4, ChemistryLiIon, ChemistryLiPo, ChemistryCustom}
		if bt.Chemistry != "" {
			valid := slices.Contains(validChemistries, bt.Chemistry)
			if !valid {
				return fmt.Errorf("invalid chemistry type: %s", bt.Chemistry)
			}
		}

		// Check that voltages are sorted ascending
		for i := 1; i < len(bt.Voltages); i++ {
			if bt.Voltages[i] <= bt.Voltages[i-1] {
				return fmt.Errorf("voltages must be in ascending order")
			}
		}

		// Check that percentages are sorted ascending
		for i := 1; i < len(bt.Percent); i++ {
			if bt.Percent[i] <= bt.Percent[i-1] {
				return fmt.Errorf("percentages must be in ascending order")
			}
		}

	}

	// Check preset name if custom not provided
	if b.CustomBatteryType == nil && b.PresetBatteryType != "" {
		found := false
		for _, preset := range PresetBatteryTypes {
			if strings.EqualFold(preset.Name, b.PresetBatteryType) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown preset battery type: %s", b.PresetBatteryType)
		}
	}

	if b.MinimumVoltageDetection < 0 {
		return fmt.Errorf("minimum voltage detection must be >= 0")
	}

	// Validate depletion estimation settings
	if b.DepletionHistoryHours < 1 || b.DepletionHistoryHours > 168 { // 1 hour to 1 week
		return fmt.Errorf("depletion history hours must be between 1 and 168")
	}

	if b.DepletionWarningHours < 0 || b.DepletionWarningHours > 720 { // 0 to 30 days
		return fmt.Errorf("depletion warning hours must be between 0 and 720")
	}

	if b.PowerSavingDischargeRatio < 0 || b.PowerSavingDischargeRatio > 1 {
		return fmt.Errorf("power saving discharge ratio must be between 0 and 1")
	}

	return nil
}

// batteryMapToStruct converts map to Battery struct
func batteryMapToStruct(m map[string]any) (any, error) {
	var s Battery
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}

	return s, nil
}
