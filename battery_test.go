package config

import (
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
	pack, err = config.GetBatteryPack(24.0)
	if err != nil {
		t.Fatalf("Failed to get battery pack: %v", err)
	}

	if pack.CellCount != 8 {
		t.Errorf("Expected 8 cells for 24V LiFePO4, got %d", pack.CellCount)
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
		{"Below minimum", 9.0, 0.0, 0.1},     // 2.25V per cell, below min 2.5V
		{"At minimum", 10.0, 0.0, 0.1},       // 2.5V per cell
		{"Above maximum", 15.0, 100.0, 0.1},  // 3.75V per cell, above max 3.65V
		{"At maximum", 14.6, 100.0, 0.1},     // 3.65V per cell
		{"Zero voltage", 0.0, 0.0, 0.1},      // Edge case
		{"Negative voltage", -5.0, 0.0, 0.1}, // Edge case (should handle gracefully)
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
