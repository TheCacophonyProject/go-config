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
	"log"
	"math"
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
	EnableVoltageReadings       bool    `mapstructure:"enable-voltage-readings"`
	Chemistry                   string  `mapstructure:"chemistry"`
	ManualCellCount             int     `mapstructure:"manual-cell-count"`
	ManuallyConfigured          bool    `mapstructure:"manually-configured"`
	MinimumVoltageDetection     float32 `mapstructure:"minimum-voltage-detection"`
	EnableDepletionEstimate     bool    `mapstructure:"enable-depletion-estimate"`
	DepletionHistoryHours       int     `mapstructure:"depletion-history-hours"`
	DepletionWarningHours       float32 `mapstructure:"depletion-warning-hours"`
	Updated                     interface{} `mapstructure:"updated,omitempty"` // Standard config timestamp
	
	// Deprecated: Migration fields for old configuration format
	PresetBatteryType  string       `mapstructure:"preset-battery-type,omitempty"`
	CustomBatteryType  *BatteryType `mapstructure:"custom-battery-type,omitempty"`
}

// DefaultBattery returns default battery configuration
func DefaultBattery() Battery {
	return Battery{
		EnableVoltageReadings:     true,
		MinimumVoltageDetection:   1.0,
		EnableDepletionEstimate:   true,
		DepletionHistoryHours:     48,
		DepletionWarningHours:     12.0,
	}
}

// NewBatteryPack creates a new battery pack with the specified chemistry and cell count
func (b *Battery) NewBatteryPack(chemistry string, cellCount int) (*BatteryPack, error) {
	batteryType, exists := ChemistryProfiles[chemistry]
	if !exists {
		return nil, fmt.Errorf("unknown chemistry: %s", chemistry)
	}
	
	return &BatteryPack{
		Type:      &batteryType,
		CellCount: cellCount,
	}, nil
}

// DetectCellCount estimates cell count for a given chemistry and voltage
func (b *Battery) DetectCellCount(chemistry string, voltage float32) int {
	batteryType, exists := ChemistryProfiles[chemistry]
	if !exists {
		return 0
	}
	
	pack := &BatteryPack{Type: &batteryType}
	return pack.DetectCellCount(voltage)
}

// BatteryType defines a single-cell battery chemistry characteristics
type BatteryType struct {
	Chemistry  string  `mapstructure:"chemistry"`
	MinVoltage float32 `mapstructure:"min-voltage"` // Per-cell minimum voltage
	MaxVoltage float32 `mapstructure:"max-voltage"` // Per-cell maximum voltage

	// Single-cell discharge curve
	Voltages []float32 `mapstructure:"voltages"`
	Percent  []float32 `mapstructure:"percent"`
	
	// Deprecated: Migration fields for old format compatibility
	Name       string  `mapstructure:"name,omitempty"`        // Old name field
	OldMinV    float32 `mapstructure:"minvoltage,omitempty"`  // Old field name
	OldMaxV    float32 `mapstructure:"maxvoltage,omitempty"`  // Old field name
}

// BatteryPack represents a complete battery pack with chemistry and cell count
type BatteryPack struct {
	Type      *BatteryType
	CellCount int
}

// DetectCellCount estimates the number of cells based on voltage reading and chemistry
func (bp *BatteryPack) DetectCellCount(voltage float32) int {
	if bp.Type == nil {
		return 0
	}
	
	// Use nominal voltage (average of min and max) for estimation
	nominalVoltage := (bp.Type.MinVoltage + bp.Type.MaxVoltage) / 2
	estimatedCells := int(voltage/nominalVoltage + 0.5) // Round to nearest integer
	
	// Validate reasonable cell count (1-24 cells)
	if estimatedCells < 1 {
		estimatedCells = 1
	} else if estimatedCells > 24 {
		estimatedCells = 24
	}
	
	return estimatedCells
}

// GetScaledMinVoltage returns the minimum voltage for the entire pack
func (bp *BatteryPack) GetScaledMinVoltage() float32 {
	if bp.Type == nil || bp.CellCount <= 0 {
		return 0
	}
	return bp.Type.MinVoltage * float32(bp.CellCount)
}

// GetScaledMaxVoltage returns the maximum voltage for the entire pack
func (bp *BatteryPack) GetScaledMaxVoltage() float32 {
	if bp.Type == nil || bp.CellCount <= 0 {
		return 0
	}
	return bp.Type.MaxVoltage * float32(bp.CellCount)
}

// VoltageToPercent converts pack voltage to percentage using scaled single-cell curve
func (bp *BatteryPack) VoltageToPercent(voltage float32) (float32, error) {
	if bp.Type == nil {
		return -1, fmt.Errorf("no battery type defined")
	}
	
	if bp.CellCount <= 0 {
		return -1, fmt.Errorf("invalid cell count: %d", bp.CellCount)
	}
	
	// Convert pack voltage to per-cell voltage
	cellVoltage := voltage / float32(bp.CellCount)
	
	// Use original single-cell curve for calculation
	voltages := bp.Type.Voltages
	percents := bp.Type.Percent
	
	// Validate curves
	if len(voltages) != len(percents) || len(voltages) == 0 {
		return -1, fmt.Errorf("invalid voltage/percent curves for %s", bp.Type.Chemistry)
	}
	
	// Handle boundary conditions
	if cellVoltage <= voltages[0] {
		return percents[0], nil
	}
	if cellVoltage >= voltages[len(voltages)-1] {
		return percents[len(percents)-1], nil
	}
	
	// Binary search for interpolation interval
	left, right := 0, len(voltages)-1
	for left < right-1 {
		mid := (left + right) / 2
		if cellVoltage < voltages[mid] {
			right = mid
		} else {
			left = mid
		}
	}
	
	// Linear interpolation
	v1, v2 := voltages[left], voltages[right]
	p1, p2 := percents[left], percents[right]
	
	if v2 == v1 {
		return p1, nil // Avoid division by zero
	}
	
	percent := p1 + (p2-p1)*(cellVoltage-v1)/(v2-v1)
	
	// Ensure result is within bounds
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}
	
	return percent, nil
}

// NormalizeCurves ensures voltage curves are properly set with backward compatibility
func (bt *BatteryType) NormalizeCurves() {
	// Set chemistry if not specified
	if bt.Chemistry == "" {
		bt.Chemistry = ChemistryCustom
	}
}

// ChemistryProfiles defines single-cell characteristics for each chemistry type
var ChemistryProfiles = map[string]BatteryType{
	ChemistryLiFePO4:  LiFePO4Chemistry,
	ChemistryLiIon:    LiIonChemistry,
	ChemistryLeadAcid: LeadAcidChemistry,
	ChemistryLiPo:     LiPoChemistry,
}

// Single-cell chemistry profiles
var LiFePO4Chemistry = BatteryType{
	Chemistry:  ChemistryLiFePO4,
	MinVoltage: 2.5,  // Per-cell minimum
	MaxVoltage: 3.65, // Per-cell maximum
	Voltages:   []float32{2.5, 3.0, 3.2, 3.22, 3.25, 3.25, 3.26, 3.3, 3.32, 3.35, 3.6},
	Percent:    []float32{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
}

var LiIonChemistry = BatteryType{
	Chemistry:  ChemistryLiIon,
	MinVoltage: 3.0,  // Per-cell minimum
	MaxVoltage: 4.2,  // Per-cell maximum
	Voltages:   []float32{3.4, 3.46, 3.51, 3.56, 3.58, 3.61, 3.62, 3.64, 3.67, 3.71, 3.76, 3.81, 3.86, 3.90, 3.93, 3.97, 4.00, 4.04, 4.07, 4.11, 4.17},
	Percent:    []float32{0.0, 5.0, 10.0, 15.0, 20.0, 25.0, 30.0, 35.0, 40.0, 45.0, 50.0, 55.0, 60.0, 65.0, 70.0, 75.0, 80.0, 85.0, 90.0, 95.0, 100.0},
}

var LeadAcidChemistry = BatteryType{
	Chemistry:  ChemistryLeadAcid,
	MinVoltage: 1.8,  // Per-cell minimum
	MaxVoltage: 2.4,  // Per-cell maximum  
	Voltages:   []float32{1.93, 1.94, 1.96, 1.98, 2.00, 2.01, 2.03, 2.05, 2.07, 2.09, 2.11},
	Percent:    []float32{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
}

var LiPoChemistry = BatteryType{
	Chemistry:  ChemistryLiPo,
	MinVoltage: 3.0,  // Per-cell minimum
	MaxVoltage: 4.2,  // Per-cell maximum
	Voltages:   []float32{3.4, 3.46, 3.51, 3.56, 3.58, 3.61, 3.62, 3.64, 3.67, 3.71, 3.76, 3.81, 3.86, 3.90, 3.93, 3.97, 4.00, 4.04, 4.07, 4.11, 4.17},
	Percent:    []float32{0.0, 5.0, 10.0, 15.0, 20.0, 25.0, 30.0, 35.0, 40.0, 45.0, 50.0, 55.0, 60.0, 65.0, 70.0, 75.0, 80.0, 85.0, 90.0, 95.0, 100.0},
}

// GetBatteryPack creates a BatteryPack from config and voltage reading
func (b *Battery) GetBatteryPack(voltage float32) (*BatteryPack, error) {
	if b.Chemistry == "" {
		return nil, fmt.Errorf("no battery chemistry specified")
	}
	
	chemistryProfile, exists := ChemistryProfiles[b.Chemistry]
	if !exists {
		return nil, fmt.Errorf("unknown battery chemistry: %s", b.Chemistry)
	}
	
	pack := &BatteryPack{
		Type: &chemistryProfile,
	}
	
	// Use manual cell count if configured, otherwise detect from voltage
	if b.ManualCellCount > 0 {
		pack.CellCount = b.ManualCellCount
	} else if voltage > 0 {
		pack.CellCount = pack.DetectCellCount(voltage)
	}
	
	return pack, nil
}

// GetChemistryProfile returns the chemistry profile for the configured chemistry
func (b *Battery) GetChemistryProfile() (*BatteryType, error) {
	if b.Chemistry == "" {
		return nil, fmt.Errorf("no battery chemistry specified")
	}
	
	chemistryProfile, exists := ChemistryProfiles[b.Chemistry]
	if !exists {
		return nil, fmt.Errorf("unknown battery chemistry: %s", b.Chemistry)
	}
	
	return &chemistryProfile, nil
}


// GetBatteryType returns the configured battery chemistry profile
// Deprecated: Use GetChemistryProfile or GetBatteryPack instead
func (b *Battery) GetBatteryType() *BatteryType {
	profile, err := b.GetChemistryProfile()
	if err != nil {
		return nil
	}
	return profile
}

// IsManuallyConfigured returns true if battery type is manually set by user
func (b *Battery) IsManuallyConfigured() bool {
	return b.ManuallyConfigured
}

// SetManualChemistry sets a manual battery chemistry override
func (b *Battery) SetManualChemistry(chemistry string) error {
	// Validate chemistry
	if _, exists := ChemistryProfiles[chemistry]; !exists {
		return fmt.Errorf("unknown battery chemistry: %s", chemistry)
	}
	
	b.Chemistry = chemistry
	b.ManuallyConfigured = true
	return nil
}

// SetManualConfiguration sets manual battery chemistry and cell count
func (b *Battery) SetManualConfiguration(chemistry string, cellCount int) error {
	// Validate chemistry
	if chemistry != "" {
		if _, exists := ChemistryProfiles[chemistry]; !exists {
			return fmt.Errorf("unknown battery chemistry: %s", chemistry)
		}
		b.Chemistry = chemistry
	}
	
	// Validate cell count
	if cellCount > 0 {
		if cellCount < 1 || cellCount > 24 {
			return fmt.Errorf("cell count must be between 1 and 24, got %d", cellCount)
		}
		b.ManualCellCount = cellCount
	}
	
	b.ManuallyConfigured = true
	return nil
}

// ClearManualConfiguration clears manual battery configuration and returns to auto-detection
func (b *Battery) ClearManualConfiguration() {
	b.ManuallyConfigured = false
	b.Chemistry = ""
	b.ManualCellCount = 0
}

// GetAvailableChemistries returns list of available battery chemistries for manual selection
func GetAvailableChemistries() []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(ChemistryProfiles))
	for chemistry, profile := range ChemistryProfiles {
		result = append(result, map[string]interface{}{
			"chemistry":   chemistry,
			"minVoltage":  profile.MinVoltage,
			"maxVoltage":  profile.MaxVoltage,
			"description": fmt.Sprintf("%s (%.1f-%.1fV per cell)", chemistry, profile.MinVoltage, profile.MaxVoltage),
		})
	}
	return result
}

// batteryValidateFunc validates battery configuration
func batteryValidateFunc(battery any) error {
	b, ok := battery.(Battery)
	if !ok {
		return fmt.Errorf("invalid battery configuration type")
	}

	// Validate chemistry if specified
	if b.Chemistry != "" {
		if _, exists := ChemistryProfiles[b.Chemistry]; !exists {
			return fmt.Errorf("unknown battery chemistry: %s", b.Chemistry)
		}
	}

	// Validate manual cell count if specified
	if b.ManualCellCount != 0 {
		if b.ManualCellCount < 1 || b.ManualCellCount > 24 {
			return fmt.Errorf("manual cell count must be between 1 and 24, got %d", b.ManualCellCount)
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

	return nil
}

// batteryMapToStruct converts map to Battery struct
func batteryMapToStruct(m map[string]any) (any, error) {
	var s Battery
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}

	// Check if migration is needed
	if needsMigration(s) {
		migrated, err := migrateBatteryConfig(s)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate battery config: %w", err)
		}
		return migrated, nil
	}

	return s, nil
}

// needsMigration checks if the battery config uses old format
func needsMigration(battery Battery) bool {
	// If new format fields are empty but old format fields exist, migration is needed
	return battery.Chemistry == "" && (battery.PresetBatteryType != "" || battery.CustomBatteryType != nil)
}

// migrateBatteryConfig converts old battery configuration format to new format
func migrateBatteryConfig(old Battery) (Battery, error) {
	new := old // Start with existing config
	
	// Map old preset types to new chemistry
	presetToChemistry := map[string]string{
		"lime":           ChemistryLiIon,    // Lime batteries are Li-Ion
		"li-ion":         ChemistryLiIon,
		"lifepo4-6v":     ChemistryLiFePO4,
		"lifepo4-12v":    ChemistryLiFePO4,
		"lifepo4-24v":    ChemistryLiFePO4,
		"lead-acid-12v":  ChemistryLeadAcid,
		"lead-acid-24v":  ChemistryLeadAcid,
	}
	
	if old.PresetBatteryType != "" {
		// Migrate from preset battery type
		chemistry, exists := presetToChemistry[old.PresetBatteryType]
		if !exists {
			return Battery{}, fmt.Errorf("unknown preset battery type for migration: %s", old.PresetBatteryType)
		}
		
		new.Chemistry = chemistry
		new.ManuallyConfigured = true // Old preset selection was manual
		
		// Estimate cell count based on old preset characteristics
		cellCount := estimateCellCountFromPreset(old.PresetBatteryType)
		if cellCount > 0 {
			new.ManualCellCount = cellCount
		}
		
		log.Printf("Migrated preset battery type '%s' to chemistry '%s' with %d cells", 
			old.PresetBatteryType, chemistry, cellCount)
		
	} else if old.CustomBatteryType != nil {
		// Migrate from custom battery type
		custom := old.CustomBatteryType
		
		// Handle old field names - copy old values to new fields if needed
		if custom.OldMinV > 0 && custom.MinVoltage == 0 {
			custom.MinVoltage = custom.OldMinV
		}
		if custom.OldMaxV > 0 && custom.MaxVoltage == 0 {
			custom.MaxVoltage = custom.OldMaxV
		}
		
		// Try to determine chemistry from custom type
		chemistry := determineChemistryFromCustomType(custom)
		new.Chemistry = chemistry
		new.ManuallyConfigured = true // Custom type was manual
		
		// Estimate cell count from voltage range
		if custom.MinVoltage > 0 && custom.MaxVoltage > 0 {
			// Determine if this looks like pack voltage or per-cell voltage
			avgVoltage := (custom.MinVoltage + custom.MaxVoltage) / 2
			if avgVoltage > 5.0 {
				// Likely pack voltage (> 5V suggests multiple cells)
				cellCount := estimateCellCountFromPackVoltage(custom.MinVoltage, custom.MaxVoltage)
				if cellCount > 0 {
					new.ManualCellCount = cellCount
					// Re-determine chemistry based on estimated pack configuration
					chemistry = determineChemistryFromPackVoltage(custom.MinVoltage, custom.MaxVoltage, cellCount)
					new.Chemistry = chemistry
				}
			} else {
				// Likely per-cell voltage (< 5V suggests single cell)
				cellCount := estimateCellCountFromVoltageRange(custom.MinVoltage, custom.MaxVoltage, chemistry)
				if cellCount > 0 {
					new.ManualCellCount = cellCount
				}
			}
		}
		
		log.Printf("Migrated custom battery type (chemistry: %s) with %d cells (minV: %.2f, maxV: %.2f)", 
			chemistry, new.ManualCellCount, custom.MinVoltage, custom.MaxVoltage)
	}
	
	// Clear old fields after migration
	new.PresetBatteryType = ""
	new.CustomBatteryType = nil
	new.Updated = nil
	
	return new, nil
}

// estimateCellCountFromPreset estimates cell count based on old preset name
func estimateCellCountFromPreset(presetName string) int {
	switch presetName {
	case "lime":
		return 10 // Lime batteries are typically 10S Li-Ion (around 37V nominal)
	case "li-ion":
		return 1  // Single cell Li-Ion
	case "lifepo4-6v":
		return 2  // 6V = 2S LiFePO4
	case "lifepo4-12v":
		return 4  // 12V = 4S LiFePO4  
	case "lifepo4-24v":
		return 8  // 24V = 8S LiFePO4
	case "lead-acid-12v":
		return 6  // 12V = 6S Lead-Acid
	case "lead-acid-24v":
		return 12 // 24V = 12S Lead-Acid
	default:
		return 0  // Unknown, let auto-detection handle it
	}
}

// determineChemistryFromCustomType tries to determine chemistry from custom battery type
func determineChemistryFromCustomType(custom *BatteryType) string {
	if custom.Chemistry != "" {
		return custom.Chemistry
	}
	
	// Try to infer from voltage characteristics
	// Check for specific voltage ranges, prioritizing more specific matches
	minV, maxV := custom.MinVoltage, custom.MaxVoltage
	
	// LiFePO4: 2.5-3.65V per cell (narrower range, check first)
	if minV >= 2.4 && maxV <= 3.7 {
		return ChemistryLiFePO4
	}
	
	// Li-Ion: 3.0-4.3V per cell  
	if minV >= 2.9 && maxV <= 4.4 {
		return ChemistryLiIon
	}
	
	// Lead-Acid: 1.8-2.4V per cell
	if minV >= 1.7 && maxV <= 2.5 {
		return ChemistryLeadAcid
	}
	
	// Default to custom if we can't determine
	return ChemistryCustom
}

// estimateCellCountFromVoltageRange estimates cell count from voltage range and chemistry
func estimateCellCountFromVoltageRange(minV, maxV float32, chemistry string) int {
	profile, exists := ChemistryProfiles[chemistry]
	if !exists {
		return 0
	}
	
	// Use average voltage for estimation
	avgVoltage := (minV + maxV) / 2
	profileAvg := (profile.MinVoltage + profile.MaxVoltage) / 2
	
	cellCount := int(avgVoltage/profileAvg + 0.5) // Round to nearest integer
	
	// Validate reasonable range
	if cellCount < 1 {
		cellCount = 1
	} else if cellCount > 24 {
		cellCount = 24
	}
	
	return cellCount
}

// estimateCellCountFromPackVoltage estimates cell count from total pack voltage range
func estimateCellCountFromPackVoltage(minPackV, maxPackV float32) int {
	// Use average pack voltage for estimation
	avgPackV := (minPackV + maxPackV) / 2
	
	// Try common cell counts and chemistries to find best match
	commonConfigs := []struct {
		chemistry string
		cellCount int
		nominalV  float32
	}{
		{ChemistryLeadAcid, 6, 11.5},   // 12V lead-acid (average ~11.5V)
		{ChemistryLeadAcid, 12, 23.0},  // 24V lead-acid
		{ChemistryLiFePO4, 4, 12.8},    // 12V LiFePO4
		{ChemistryLiFePO4, 8, 25.6},    // 24V LiFePO4
		{ChemistryLiIon, 3, 11.1},      // 3S Li-Ion
		{ChemistryLiIon, 4, 14.8},      // 4S Li-Ion
		{ChemistryLiIon, 7, 25.9},      // 7S Li-Ion
		{ChemistryLiIon, 8, 29.6},      // 8S Li-Ion
		{ChemistryLiIon, 10, 37.0},     // 10S Li-Ion (lime)
	}
	
	bestMatch := 0
	bestDiff := float32(999.0)
	
	for _, config := range commonConfigs {
		diff := float32(math.Abs(float64(avgPackV - config.nominalV)))
		if diff < bestDiff {
			bestDiff = diff
			bestMatch = config.cellCount
		}
	}
	
	// Only return if the match is reasonable (within 20% of nominal)
	if bestDiff < avgPackV*0.2 {
		return bestMatch
	}
	
	return 0
}

// determineChemistryFromPackVoltage determines chemistry from pack voltage and cell count
func determineChemistryFromPackVoltage(minPackV, maxPackV float32, cellCount int) string {
	if cellCount == 0 {
		return ChemistryCustom
	}
	
	// Calculate per-cell voltage
	avgPerCell := (minPackV + maxPackV) / 2 / float32(cellCount)
	
	// Determine chemistry based on per-cell voltage
	if avgPerCell >= 2.9 && avgPerCell <= 4.4 {
		return ChemistryLiIon
	} else if avgPerCell >= 2.4 && avgPerCell <= 3.7 {
		return ChemistryLiFePO4
	} else if avgPerCell >= 1.7 && avgPerCell <= 2.5 {
		return ChemistryLeadAcid
	}
	
	return ChemistryCustom
}
