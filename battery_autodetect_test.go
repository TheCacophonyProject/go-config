package config

import (
	"testing"
)

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
			name:              "3.86V should detect Li-ion 1 cell, not Lead-acid 2 cells",
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
			name:              "12.0V should detect Li-ion 3 cells",
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