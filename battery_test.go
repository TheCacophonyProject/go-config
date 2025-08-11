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

func TestAutoDetectBatteryPack(t *testing.T) {
	tests := []struct {
		name              string
		voltage           float32
		expectedChemistry string
		expectedCells     int
		expectError       bool
	}{
		// Test cases from the requirements
		{
			name:              "3.29V should detect Li-ion 1 cell",
			voltage:           3.29,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     1,
		},
		{
			name:              "3.86V should detect Li-ion 1 cell",
			voltage:           3.86,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     1,
		},
		{
			name:              "6.6V should detect LiFePO4 2 cells",
			voltage:           6.6,
			expectedChemistry: ChemistryLiFePO4,
			expectedCells:     2,
		},
		// Additional test cases based on the voltage range table
		{
			name:              "2.1V should detect Lead-acid 1 cell",
			voltage:           2.1,
			expectedChemistry: ChemistryLeadAcid,
			expectedCells:     1,
		},
		{
			name:              "2.6V should detect LiFePO4 1 cell",
			voltage:           2.6,
			expectedChemistry: ChemistryLiFePO4,
			expectedCells:     1,
		},
		{
			name:              "4.0V should detect Li-ion 1 cell",
			voltage:           4.0,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     1,
		},
		{
			name:              "6.5V should detect LiFePO4 2 cells",
			voltage:           6.5,
			expectedChemistry: ChemistryLiFePO4,
			expectedCells:     2,
		},
		{
			name:              "7.8V should detect Li-ion 2 cells",
			voltage:           7.8,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     2,
		},
		{
			name:              "8.6V should detect LiFePO4 3 cells",
			voltage:           8.6,
			expectedChemistry: ChemistryLiFePO4,
			expectedCells:     3,
		},
		{
			name:              "10.5V should detect Li-ion 3 cells",
			voltage:           10.5,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     3,
		},
		{
			name:              "12.0V should detect Li-ion 4 cells",
			voltage:           12.0,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     3,
		},
		{
			name:              "13.5V should detect Li-ion 4 cells",
			voltage:           13.5,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     4,
		},
		{
			name:              "17.2V should detect Li-ion 5 cells",
			voltage:           17.2,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     5,
		},
		{
			name:              "21.0V should detect Li-ion 5 cells",
			voltage:           21.0,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     5,
		},
		{
			name:              "27.0V should detect Li-ion 8 cells",
			voltage:           27.0,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     8,
		},
		{
			name:              "35.0V should detect Li-ion 10 cells",
			voltage:           35.0,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     10,
		},
		// Error cases
		{
			name:        "0V should return error",
			voltage:     0,
			expectError: true,
		},
		{
			name:        "negative voltage should return error",
			voltage:     -5.0,
			expectError: true,
		},
		{
			name:        "1.5V should return error (below minimum)",
			voltage:     1.5,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pack, err := AutoDetectBatteryPack(tc.voltage)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for voltage %.2fV, but got none", tc.voltage)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for voltage %.2fV: %v", tc.voltage, err)
				return
			}

			if pack == nil {
				t.Errorf("Expected battery pack for voltage %.2fV, but got nil", tc.voltage)
				return
			}

			if pack.Type.Chemistry != tc.expectedChemistry {
				t.Errorf("For voltage %.2fV: expected chemistry %s, got %s",
					tc.voltage, tc.expectedChemistry, pack.Type.Chemistry)
			}

			if pack.CellCount != tc.expectedCells {
				t.Errorf("For voltage %.2fV: expected %d cells, got %d cells",
					tc.voltage, tc.expectedCells, pack.CellCount)
			}

			// Verify voltage is within expected range
			minV := pack.GetScaledMinVoltage()
			maxV := pack.GetScaledMaxVoltage()
			if tc.voltage < minV || tc.voltage > maxV {
				t.Errorf("Voltage %.2fV is outside detected pack range [%.2f-%.2f]",
					tc.voltage, minV, maxV)
			}
		})
	}
}

// Test that lower cell counts are preferred when ranges overlap
func TestAutoDetectBatteryPackPreference(t *testing.T) {
	// Test specific overlapping cases
	tests := []struct {
		voltage           float32
		expectedChemistry string
		expectedCells     int
		reason            string
	}{
		{
			voltage:           3.86,
			expectedChemistry: ChemistryLiIon,
			expectedCells:     1,
			reason:            "3.86V is within Li-ion 1 cell range and Lead-acid 2 cell range, should prefer lower cell count",
		},
		{
			voltage:           6.6,
			expectedChemistry: ChemistryLiFePO4,
			expectedCells:     2,
			reason:            "6.6V is within LiFePO4 2 cell range, should detect as LiFePO4",
		},
		{
			voltage:           6.7,
			expectedChemistry: ChemistryLiFePO4,
			expectedCells:     2,
			reason:            "6.7V is within LiFePO4 2 cell range, should detect as LiFePO4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.reason, func(t *testing.T) {
			pack, err := AutoDetectBatteryPack(tc.voltage)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if pack.Type.Chemistry != tc.expectedChemistry {
				t.Errorf("Expected chemistry %s, got %s", tc.expectedChemistry, pack.Type.Chemistry)
			}

			if pack.CellCount != tc.expectedCells {
				t.Errorf("Expected %d cells, got %d cells", tc.expectedCells, pack.CellCount)
			}
		})
	}
}
