package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBatteryPackDetectCellCount(t *testing.T) {
	// Test LiFePO4 chemistry
	config := Battery{
		Chemistry: ChemistryLiFePO4,
	}
	
	// Test 12V LiFePO4 (4 cells)
	pack, err := config.GetBatteryPack(13.0)
	if err != nil {
		t.Fatalf("Failed to get battery pack: %v", err)
	}
	
	if pack.CellCount != 4 {
		t.Errorf("Expected 4 cells for 13V LiFePO4, got %d", pack.CellCount)
	}
	
	// Test 24V LiFePO4 (8 cells)
	pack, err = config.GetBatteryPack(26.0)
	if err != nil {
		t.Fatalf("Failed to get battery pack: %v", err)
	}
	
	if pack.CellCount != 8 {
		t.Errorf("Expected 8 cells for 26V LiFePO4, got %d", pack.CellCount)
	}
}

func TestBatteryPackVoltageToPercent(t *testing.T) {
	// Test LiFePO4 chemistry
	config := Battery{
		Chemistry: ChemistryLiFePO4,
	}
	
	// Test 12V LiFePO4 (4 cells)
	pack, err := config.GetBatteryPack(13.0)
	if err != nil {
		t.Fatalf("Failed to get battery pack: %v", err)
	}
	
	// Test voltage to percentage conversion
	percent, err := pack.VoltageToPercent(13.0) // 3.25V per cell (mid-range)
	if err != nil {
		t.Fatalf("Failed to convert voltage to percent: %v", err)
	}
	
	if percent < 40 || percent > 60 {
		t.Errorf("Expected percentage around 50%% for mid-range voltage, got %.1f%%", percent)
	}
	
	// Test minimum voltage
	minPercent, err := pack.VoltageToPercent(pack.GetScaledMinVoltage())
	if err != nil {
		t.Fatalf("Failed to convert min voltage to percent: %v", err)
	}
	
	if minPercent != 0 {
		t.Errorf("Expected 0%% for minimum voltage, got %.1f%%", minPercent)
	}
	
	// Test maximum voltage
	maxPercent, err := pack.VoltageToPercent(pack.GetScaledMaxVoltage())
	if err != nil {
		t.Fatalf("Failed to convert max voltage to percent: %v", err)
	}
	
	if maxPercent != 100 {
		t.Errorf("Expected 100%% for maximum voltage, got %.1f%%", maxPercent)
	}
}

func TestLiIonChemistry(t *testing.T) {
	// Test Li-Ion chemistry
	config := Battery{
		Chemistry: ChemistryLiIon,
	}
	
	// Test 14.8V Li-Ion (4 cells)
	pack, err := config.GetBatteryPack(14.8)
	if err != nil {
		t.Fatalf("Failed to get battery pack: %v", err)
	}
	
	if pack.CellCount != 4 {
		t.Errorf("Expected 4 cells for 14.8V Li-Ion, got %d", pack.CellCount)
	}
	
	// Test voltage conversion
	percent, err := pack.VoltageToPercent(14.8) // 3.7V per cell (nominal)
	if err != nil {
		t.Fatalf("Failed to convert voltage to percent: %v", err)
	}
	
	if percent < 40 || percent > 60 {
		t.Errorf("Expected percentage around 50%% for nominal voltage, got %.1f%%", percent)
	}
}

func TestInvalidChemistry(t *testing.T) {
	config := Battery{
		Chemistry: "invalid-chemistry",
	}
	
	_, err := config.GetBatteryPack(12.0)
	if err == nil {
		t.Error("Expected error for invalid chemistry, got nil")
	}
}

func TestCellDetectionBoundaryConditions(t *testing.T) {
	// Test edge cases for cell count detection
	testCases := []struct {
		name          string
		chemistry     string
		voltage       float32
		expectedCells int
		description   string
	}{
		// LiFePO4 boundary cases (nominal 3.25V per cell)
		{"LiFePO4 1-2 cell boundary", "lifepo4", 4.875, 2, "4.875V is exactly 1.5x nominal, should round to 2"},
		{"LiFePO4 2-3 cell boundary", "lifepo4", 8.125, 3, "8.125V is exactly 2.5x nominal, should round to 3"},
		{"LiFePO4 very low voltage", "lifepo4", 0.5, 1, "Very low voltage should default to 1 cell"},
		{"LiFePO4 very high voltage", "lifepo4", 100.0, 24, "Very high voltage should cap at 24 cells"},
		
		// Li-Ion boundary cases (nominal 3.7V per cell)
		{"Li-Ion 7-8 cell boundary", "li-ion", 27.75, 8, "27.75V is exactly 7.5x nominal, should round to 8"},
		{"Li-Ion 8-9 cell boundary", "li-ion", 31.45, 9, "31.45V is exactly 8.5x nominal, should round to 9"},
		
		// Lead-Acid boundary cases (nominal 2.1V per cell)
		{"Lead-Acid 5-6 cell boundary", "lead-acid", 11.55, 6, "11.55V is exactly 5.5x nominal, should round to 6"},
		{"Lead-Acid typical 12V", "lead-acid", 12.6, 6, "12.6V typical lead-acid should be 6 cells"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultBattery()
			cellCount := config.DetectCellCount(tc.chemistry, tc.voltage)
			if cellCount != tc.expectedCells {
				t.Errorf("%s: expected %d cells, got %d (%s)",
					tc.name, tc.expectedCells, cellCount, tc.description)
			}
		})
	}
}

func TestManualChemistryConfiguration(t *testing.T) {
	config := DefaultBattery()
	
	// Test setting manual chemistry
	err := config.SetManualChemistry("lifepo4")
	if err != nil {
		t.Fatalf("Failed to set manual chemistry: %v", err)
	}
	
	if !config.IsManuallyConfigured() {
		t.Error("Battery should be manually configured after SetManualChemistry")
	}
	
	if config.Chemistry != "lifepo4" {
		t.Errorf("Expected chemistry 'lifepo4', got '%s'", config.Chemistry)
	}
	
	// Test clearing manual configuration
	config.ClearManualConfiguration()
	if config.IsManuallyConfigured() {
		t.Error("Battery should not be manually configured after ClearManualConfiguration")
	}
	
	if config.Chemistry != "" {
		t.Errorf("Expected empty chemistry after clear, got '%s'", config.Chemistry)
	}
	
	// Test setting invalid chemistry
	err = config.SetManualChemistry("invalid-chemistry")
	if err == nil {
		t.Error("Expected error for invalid chemistry, got nil")
	}
}

func TestVoltageToPercentBoundaries(t *testing.T) {
	// Test voltage to percentage conversion at boundaries
	config := Battery{
		Chemistry: ChemistryLiFePO4,
	}
	
	pack, err := config.GetBatteryPack(13.0) // 4-cell LiFePO4
	if err != nil {
		t.Fatalf("Failed to get battery pack: %v", err)
	}
	
	testCases := []struct {
		name            string
		voltage         float32
		expectedPercent float32
		tolerance       float32
	}{
		{"Below minimum", 9.0, 0.0, 0.1},      // 2.25V per cell, below min 2.5V
		{"At minimum", 10.0, 0.0, 0.1},        // 2.5V per cell
		{"Above maximum", 15.0, 100.0, 0.1},   // 3.75V per cell, above max 3.65V
		{"At maximum", 14.6, 100.0, 0.1},      // 3.65V per cell
		{"Zero voltage", 0.0, 0.0, 0.1},       // Edge case
		{"Negative voltage", -5.0, 0.0, 0.1},  // Edge case (should handle gracefully)
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			percent, err := pack.VoltageToPercent(tc.voltage)
			if err != nil {
				t.Fatalf("Failed to convert voltage to percent: %v", err)
			}
			
			if percent < tc.expectedPercent-tc.tolerance || percent > tc.expectedPercent+tc.tolerance {
				t.Errorf("For voltage %.1fV, expected %.1f%% (±%.1f), got %.1f%%",
					tc.voltage, tc.expectedPercent, tc.tolerance, percent)
			}
		})
	}
}

func TestGetAvailableChemistries(t *testing.T) {
	chemistries := GetAvailableChemistries()
	
	// Should have at least the standard chemistries
	if len(chemistries) < 4 {
		t.Errorf("Expected at least 4 chemistries, got %d", len(chemistries))
	}
	
	// Check that each chemistry has required fields
	requiredChemistries := []string{"lifepo4", "li-ion", "lead-acid", "lipo"}
	foundChemistries := make(map[string]bool)
	
	for _, chem := range chemistries {
		chemistry, ok := chem["chemistry"].(string)
		if !ok {
			t.Error("Chemistry missing 'chemistry' field")
			continue
		}
		foundChemistries[chemistry] = true
		
		if _, ok := chem["minVoltage"].(float32); !ok {
			t.Errorf("Chemistry %s missing minVoltage", chemistry)
		}
		
		if _, ok := chem["maxVoltage"].(float32); !ok {
			t.Errorf("Chemistry %s missing maxVoltage", chemistry)
		}
		
		if _, ok := chem["description"].(string); !ok {
			t.Errorf("Chemistry %s missing description", chemistry)
		}
	}
	
	// Verify all required chemistries are present
	for _, req := range requiredChemistries {
		if !foundChemistries[req] {
			t.Errorf("Required chemistry '%s' not found", req)
		}
	}
}

func TestBatteryMigration(t *testing.T) {
	// Test migration from preset-battery-type format
	t.Run("Migrate lime preset", func(t *testing.T) {
		old := Battery{
			EnableVoltageReadings:   true,
			ManuallyConfigured:      true,
			MinimumVoltageDetection: 1.0,
			PresetBatteryType:       "lime",
			// Chemistry is empty - should trigger migration
		}
		
		if !needsMigration(old) {
			t.Error("Expected migration to be needed for lime preset")
		}
		
		migrated, err := migrateBatteryConfig(old)
		if err != nil {
			t.Fatalf("Migration failed: %v", err)
		}
		
		if migrated.Chemistry != ChemistryLiIon {
			t.Errorf("Expected chemistry %s, got %s", ChemistryLiIon, migrated.Chemistry)
		}
		
		if migrated.ManualCellCount != 10 {
			t.Errorf("Expected 10 cells for lime battery, got %d", migrated.ManualCellCount)
		}
		
		if !migrated.ManuallyConfigured {
			t.Error("Expected manually configured to remain true")
		}
		
		// Old fields should be cleared
		if migrated.PresetBatteryType != "" {
			t.Error("Expected PresetBatteryType to be cleared after migration")
		}
	})
	
	t.Run("Migrate LiFePO4 12V preset", func(t *testing.T) {
		old := Battery{
			PresetBatteryType: "lifepo4-12v",
		}
		
		migrated, err := migrateBatteryConfig(old)
		if err != nil {
			t.Fatalf("Migration failed: %v", err)
		}
		
		if migrated.Chemistry != ChemistryLiFePO4 {
			t.Errorf("Expected chemistry %s, got %s", ChemistryLiFePO4, migrated.Chemistry)
		}
		
		if migrated.ManualCellCount != 4 {
			t.Errorf("Expected 4 cells for 12V LiFePO4, got %d", migrated.ManualCellCount)
		}
	})
	
	t.Run("Migrate custom battery type", func(t *testing.T) {
		customType := &BatteryType{
			Chemistry:  ChemistryLiIon,
			MinVoltage: 3.0,
			MaxVoltage: 4.2,
		}
		
		old := Battery{
			CustomBatteryType: customType,
		}
		
		migrated, err := migrateBatteryConfig(old)
		if err != nil {
			t.Fatalf("Migration failed: %v", err)
		}
		
		if migrated.Chemistry != ChemistryLiIon {
			t.Errorf("Expected chemistry %s, got %s", ChemistryLiIon, migrated.Chemistry)
		}
		
		// Should detect single cell from voltage range
		if migrated.ManualCellCount != 1 {
			t.Errorf("Expected 1 cell for single-cell voltage range, got %d", migrated.ManualCellCount)
		}
	})
	
	t.Run("No migration needed for new format", func(t *testing.T) {
		newFormat := Battery{
			Chemistry:       ChemistryLiFePO4,
			ManualCellCount: 8,
		}
		
		if needsMigration(newFormat) {
			t.Error("Should not need migration for new format")
		}
	})
	
	t.Run("Unknown preset fails migration", func(t *testing.T) {
		old := Battery{
			PresetBatteryType: "unknown-battery",
		}
		
		_, err := migrateBatteryConfig(old)
		if err == nil {
			t.Error("Expected error for unknown preset battery type")
		}
	})
}

func TestCellCountEstimation(t *testing.T) {
	tests := []struct {
		preset   string
		expected int
	}{
		{"lime", 10},
		{"li-ion", 1},
		{"lifepo4-6v", 2},
		{"lifepo4-12v", 4},
		{"lifepo4-24v", 8},
		{"lead-acid-12v", 6},
		{"lead-acid-24v", 12},
		{"unknown", 0},
	}
	
	for _, test := range tests {
		t.Run(test.preset, func(t *testing.T) {
			result := estimateCellCountFromPreset(test.preset)
			if result != test.expected {
				t.Errorf("Expected %d cells for %s, got %d", test.expected, test.preset, result)
			}
		})
	}
}

func TestChemistryDetection(t *testing.T) {
	tests := []struct {
		name     string
		battery  *BatteryType
		expected string
	}{
		{
			name: "Explicit chemistry",
			battery: &BatteryType{
				Chemistry: ChemistryLiFePO4,
			},
			expected: ChemistryLiFePO4,
		},
		{
			name: "Li-Ion voltage range",
			battery: &BatteryType{
				MinVoltage: 3.0,
				MaxVoltage: 4.2,
			},
			expected: ChemistryLiIon,
		},
		{
			name: "LiFePO4 voltage range",
			battery: &BatteryType{
				MinVoltage: 2.5,
				MaxVoltage: 3.6,
			},
			expected: ChemistryLiFePO4,
		},
		{
			name: "Lead-Acid voltage range",
			battery: &BatteryType{
				MinVoltage: 1.8,
				MaxVoltage: 2.3,
			},
			expected: ChemistryLeadAcid,
		},
		{
			name: "Unknown voltage range",
			battery: &BatteryType{
				MinVoltage: 5.0,
				MaxVoltage: 6.0,
			},
			expected: ChemistryCustom,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := determineChemistryFromCustomType(test.battery)
			if result != test.expected {
				t.Errorf("Expected chemistry %s, got %s", test.expected, result)
			}
		})
	}
}

func TestUnmarshalWithMigration(t *testing.T) {
	// Create a temporary config file with old format
	configDir := t.TempDir()
	
	// Write config with old battery format
	configContent := `
[battery]
preset-battery-type = "lime"
manually-configured = true
enable-voltage-readings = true
minimum-voltage-detection = 1.0
enable-depletion-estimate = true
depletion-history-hours = 48
depletion-warning-hours = 12.0
power-saving-discharge-ratio = 0.3
`
	
	err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	
	// Load config using the actual Config.Unmarshal method (like tc2-hat-attiny does)
	config, err := New(configDir)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	
	// This should trigger migration
	battery := DefaultBattery()
	err = config.Unmarshal(BatteryKey, &battery)
	if err != nil {
		t.Fatalf("Failed to unmarshal battery config: %v", err)
	}
	
	// Verify migration occurred
	if battery.Chemistry != ChemistryLiIon {
		t.Errorf("Expected chemistry %s after migration, got %s", ChemistryLiIon, battery.Chemistry)
	}
	
	if battery.ManualCellCount != 10 {
		t.Errorf("Expected 10 cells after migration, got %d", battery.ManualCellCount)
	}
	
	if !battery.ManuallyConfigured {
		t.Error("Expected manually configured to remain true")
	}
	
	// Verify old fields are cleared
	if battery.PresetBatteryType != "" {
		t.Error("Expected PresetBatteryType to be cleared after migration")
	}
	
	t.Logf("Migration successful: lime -> chemistry=%s, cells=%d", 
		battery.Chemistry, battery.ManualCellCount)
}

func TestUnmarshalWithCustomBatteryMigration(t *testing.T) {
	// Create a temporary config file with old custom battery format
	configDir := t.TempDir()
	
	// Write config with old custom battery format (matching the error from the logs)
	configContent := `
[battery]
enable-voltage-readings = true
manually-configured = true
minimum-voltage-detection = 1.0
updated = "2025-07-24T17:45:35.395890664+12:00"

[battery.custom-battery-type]
name = "custom-12v"
chemistry = ""
minvoltage = 9.0
maxvoltage = 14.0
voltages = [11.59, 11.63, 11.76, 11.87, 11.97, 12.07, 12.18, 12.29, 12.41, 12.53, 12.64]
percent = [0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100]
`
	
	err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	
	// Load config using the actual Config.Unmarshal method
	config, err := New(configDir)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	
	// This should trigger migration
	battery := DefaultBattery()
	err = config.Unmarshal(BatteryKey, &battery)
	if err != nil {
		t.Fatalf("Failed to unmarshal battery config: %v", err)
	}
	
	// Verify migration occurred
	if battery.Chemistry == "" {
		t.Error("Expected chemistry to be set after migration")
	}
	
	if battery.ManualCellCount == 0 {
		t.Error("Expected cell count to be set after migration")
	}
	
	if !battery.ManuallyConfigured {
		t.Error("Expected manually configured to remain true")
	}
	
	// Verify old fields are cleared
	if battery.CustomBatteryType != nil {
		t.Error("Expected CustomBatteryType to be cleared after migration")
	}
	
	t.Logf("Custom battery migration successful: chemistry=%s, cells=%d", 
		battery.Chemistry, battery.ManualCellCount)
}