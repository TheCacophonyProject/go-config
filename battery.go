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
		defaultValue: func() any {
			return DefaultBattery()
		},
		pointerValue: func() any {
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
	Updated                     any `mapstructure:"updated,omitempty"` // Standard config timestamp
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
	Chemistry       string  `mapstructure:"chemistry"`
	MinVoltage      float32 `mapstructure:"min-voltage"`     // Per-cell minimum voltage
	MaxVoltage      float32 `mapstructure:"max-voltage"`     // Per-cell maximum voltage

	// Single-cell discharge curve
	Voltages []float32 `mapstructure:"voltages"`
	Percent  []float32 `mapstructure:"percent"`
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
	
	// Use nominal voltage if specified, otherwise fall back to average of min and max
	nominalVoltage := (bp.Type.MinVoltage + bp.Type.MaxVoltage) / 2
	estimatedCells := int(voltage/nominalVoltage + 0.5) // Round to nearest integer
	
	// Validate reasonable cell count (1-24 cells)
	if estimatedCells < 1 {
		estimatedCells = 1
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
	Chemistry:      ChemistryLiFePO4,
	MinVoltage:     2.5,  // Per-cell minimum
	MaxVoltage:     3.4, // Per-cell maximum (expanded to handle 6.92V ÷ 2 = 3.46V)
	Voltages:       []float32{2.5, 3.0, 3.2, 3.22, 3.25, 3.26, 3.27, 3.3, 3.32, 3.35,3.4},
	Percent:        []float32{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
}

var LiIonChemistry = BatteryType{
	Chemistry:      ChemistryLiIon,
	MinVoltage:     2.5,  // Per-cell minimum 
	MaxVoltage:     3.4, // Per-cell maximum (typical Li-ion range)
	Voltages:       []float32{2.5, 3.0, 3.2, 3.22, 3.25, 3.26, 3.27, 3.3, 3.32, 3.35, 3.4},
	Percent:        []float32{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
}

var LeadAcidChemistry = BatteryType{
	Chemistry:      ChemistryLeadAcid,
	MinVoltage:     1.94,  // Per-cell minimum
	MaxVoltage:     2.15,  // Per-cell maximum
	Voltages:       []float32{1.94, 1.95, 1.97, 1.99, 2.02, 2.04, 2.07, 2.09, 2.11, 2.13, 2.15},
	Percent:        []float32{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
}

var LiPoChemistry = BatteryType{
	Chemistry:      ChemistryLiPo,
	MinVoltage:     3.27,  // Per-cell minimum
	MaxVoltage:     4.2, // Per-cell maximum
	Voltages:       []float32{3.27, 3.69, 3.73, 3.77, 3.8, 3.84, 3.87, 3.95, 4.02, 4.11, 4.2},
	Percent:        []float32{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
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
	
	// Validate and set cell count
	if cellCount > 0 {
		if cellCount < 1 || cellCount > 24 {
			return fmt.Errorf("cell count must be between 1 and 24, got %d", cellCount)
		}
		b.ManualCellCount = cellCount
	} else if cellCount == 0 {
		// Explicitly set to 0 for auto-detection
		b.ManualCellCount = 0
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

// AutoDetectBatteryPack detects battery chemistry and cell count based on voltage
// It first checks against the authoritative voltage table, then falls back to range matching
func AutoDetectBatteryPack(voltage float32) (*BatteryPack, error) {
	if voltage <= 0 {
		return nil, fmt.Errorf("invalid voltage for detection: %.2fV", voltage)
	}
	
	// First priority: Check against the authoritative voltage table
	if pack := checkVoltageTable(voltage); pack != nil {
		return pack, nil
	}
	
	// Second priority: Fall back to range matching with lower cell count preference
	return fallbackDetection(voltage)
}

// checkVoltageTable checks voltage against the authoritative voltage range table
func checkVoltageTable(voltage float32) *BatteryPack {
	// Voltage table ranges (authoritative source)
	type tableEntry struct {
		chemistry string
		cells     int
	}
	
	voltageTable := []tableEntry{
		{ChemistryLeadAcid, 1},
		{ChemistryLiFePO4, 1},
		{ChemistryLiIon, 1},
		{ChemistryLiFePO4, 2},
		{ChemistryLiIon, 2},
		{ChemistryLiFePO4, 3},
		{ChemistryLiIon, 3},
		{ChemistryLeadAcid,6},
		{ChemistryLiIon, 4},
		{ChemistryLiIon,5},
		{ChemistryLiIon,6},
		{ChemistryLiIon,8},
		{ChemistryLiIon,10},
		{ChemistryLiIon,12},
	}
	
	// Check if voltage falls into any table range
	// For overlapping ranges, the first match in the table wins (table is ordered by preference)
	for _, entry := range voltageTable {
		chemProfile, exists := ChemistryProfiles[entry.chemistry]
		if !exists {
			continue // Skip unknown chemistries
		}
		minVoltage := chemProfile.MinVoltage * float32(entry.cells)
		maxVoltage := chemProfile.MaxVoltage * float32(entry.cells)
		if voltage >= minVoltage && voltage <= maxVoltage {
			return &BatteryPack{
				Type:      &chemProfile,
				CellCount: entry.cells,
			}
		}
	}
	
	return nil // No table match found
}

// fallbackDetection provides fallback detection when voltage doesn't match the table
func fallbackDetection(voltage float32) (*BatteryPack, error) {
	type voltageRange struct {
		min       float32
		max       float32
		chemistry string
		cells     int
	}
	
	// Build ranges from chemistry profiles, iterating from 1 to 10 cells
	var ranges []voltageRange
	maxCells := 10
	
	// Iterate through cell counts from 1 to maxCells
	for cells := 1; cells <= maxCells; cells++ {
		// Check each chemistry for this cell count
		for chemName, chem := range ChemistryProfiles {
			minV := chem.MinVoltage * float32(cells)
			maxV := chem.MaxVoltage * float32(cells)
			
			ranges = append(ranges, voltageRange{
				min:       minV,
				max:       maxV,
				chemistry: chemName,
				cells:     cells,
			})
		}
	}
	
	// Find all matching ranges for the given voltage
	var matches []voltageRange
	for _, r := range ranges {
		if voltage >= r.min && voltage <= r.max {
			matches = append(matches, r)
		}
	}
	
	if len(matches) == 0 {
		return nil, fmt.Errorf("no battery chemistry matches voltage %.2fV", voltage)
	}
	
	// Prefer lower cell count as tiebreaker
	bestMatch := matches[0]
	for _, match := range matches {
		if match.cells < bestMatch.cells {
			bestMatch = match
		}
	}
	
	chemProfile, exists := ChemistryProfiles[bestMatch.chemistry]
	if !exists {
		return nil, fmt.Errorf("chemistry profile not found: %s", bestMatch.chemistry)
	}
	
	return &BatteryPack{
		Type:      &chemProfile,
		CellCount: bestMatch.cells,
	}, nil
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
	return s, nil
}

