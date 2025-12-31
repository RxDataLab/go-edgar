package edgar

import (
	"testing"
)

func TestConceptMappings(t *testing.T) {
	// Test that concept mappings are loaded
	labels := GetAllStandardizedLabels()
	if len(labels) == 0 {
		t.Fatal("No concept mappings loaded")
	}

	t.Logf("Loaded %d standardized labels", len(labels))

	// Test reverse lookup: XBRL concept -> standardized label
	tests := []struct {
		xbrlConcept   string
		expectedLabel string
	}{
		{
			xbrlConcept:   "us-gaap:CashAndCashEquivalentsAtCarryingValue",
			expectedLabel: "Cash and Cash Equivalents",
		},
		{
			xbrlConcept:   "us-gaap:ResearchAndDevelopmentExpense",
			expectedLabel: "Research and Development Expense",
		},
		{
			xbrlConcept:   "us-gaap:GeneralAndAdministrativeExpense",
			expectedLabel: "General and Administrative Expense",
		},
		{
			xbrlConcept:   "us-gaap:LongTermDebt",
			expectedLabel: "Long-Term Debt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.xbrlConcept, func(t *testing.T) {
			label := GetStandardizedLabel(tt.xbrlConcept)
			if label != tt.expectedLabel {
				t.Errorf("GetStandardizedLabel(%q) = %q, want %q",
					tt.xbrlConcept, label, tt.expectedLabel)
			}

			if !HasMapping(tt.xbrlConcept) {
				t.Errorf("HasMapping(%q) = false, want true", tt.xbrlConcept)
			}
		})
	}

	// Test forward lookup: standardized label -> XBRL concepts
	concepts, err := GetConceptsForLabel("Cash and Cash Equivalents")
	if err != nil {
		t.Fatalf("GetConceptsForLabel failed: %v", err)
	}

	if len(concepts) == 0 {
		t.Error("Expected at least one concept for 'Cash and Cash Equivalents'")
	}

	t.Logf("Cash and Cash Equivalents maps to %d concepts: %v", len(concepts), concepts)

	// Test unknown concept
	unknownLabel := GetStandardizedLabel("us-gaap:ThisDoesNotExist")
	if unknownLabel != "" {
		t.Errorf("GetStandardizedLabel(unknown) = %q, want empty string", unknownLabel)
	}

	// Test unknown label
	_, err = GetConceptsForLabel("This Label Does Not Exist")
	if err == nil {
		t.Error("GetConceptsForLabel(unknown) should return error")
	}
}

func TestConceptMappingCaseInsensitive(t *testing.T) {
	// Test case-insensitive matching
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "us-gaap:CashAndCashEquivalentsAtCarryingValue",
			expected: "Cash and Cash Equivalents",
		},
		{
			input:    "US-GAAP:CASHANDCASHEQUIVALENTSATCARRYINGVALUE", // All caps
			expected: "Cash and Cash Equivalents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			label := GetStandardizedLabel(tt.input)
			if label != tt.expected {
				t.Errorf("GetStandardizedLabel(%q) = %q, want %q",
					tt.input, label, tt.expected)
			}
		})
	}
}
