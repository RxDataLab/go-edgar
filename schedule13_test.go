package edgar

import (
	"os"
	"testing"
)

func TestParseSchedule13D(t *testing.T) {
	// Test with edgartools reference data
	data, err := os.ReadFile("/home/nick/projects/port-edgartools/edgartools/tests/data/beneficial_ownership/schedule13d.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	filing, err := ParseSchedule13D(data)
	if err != nil {
		t.Fatalf("Failed to parse Schedule 13D: %v", err)
	}

	// Validate basic fields
	if filing.FormType != "SCHEDULE 13D" {
		t.Errorf("Expected FormType 'SCHEDULE 13D', got '%s'", filing.FormType)
	}

	if filing.IssuerCIK != "0001422142" {
		t.Errorf("Expected IssuerCIK '0001422142', got '%s'", filing.IssuerCIK)
	}

	if filing.IssuerName != "Aadi Bioscience, Inc." {
		t.Errorf("Expected IssuerName 'Aadi Bioscience, Inc.', got '%s'", filing.IssuerName)
	}

	if filing.IssuerCUSIP != "00032Q104" {
		t.Errorf("Expected IssuerCUSIP '00032Q104', got '%s'", filing.IssuerCUSIP)
	}

	if filing.SecurityTitle != "Common stock, par value $0.0001 per share" {
		t.Errorf("Expected SecurityTitle 'Common stock, par value $0.0001 per share', got '%s'", filing.SecurityTitle)
	}

	if filing.DateOfEvent != "12/31/2024" {
		t.Errorf("Expected DateOfEvent '12/31/2024', got '%s'", filing.DateOfEvent)
	}

	if !filing.PreviouslyFiled {
		t.Error("Expected PreviouslyFiled to be true")
	}

	// Validate reporting persons
	if len(filing.ReportingPersons) != 2 {
		t.Fatalf("Expected 2 reporting persons, got %d", len(filing.ReportingPersons))
	}

	// First reporting person: BML Investment Partners, L.P.
	p1 := filing.ReportingPersons[0]
	if p1.CIK != "0001373604" {
		t.Errorf("Expected person 1 CIK '0001373604', got '%s'", p1.CIK)
	}
	if p1.Name != "BML Investment Partners, L.P." {
		t.Errorf("Expected person 1 name 'BML Investment Partners, L.P.', got '%s'", p1.Name)
	}
	if p1.AggregateAmountOwned != 2100000 {
		t.Errorf("Expected person 1 aggregate 2100000, got %d", p1.AggregateAmountOwned)
	}
	if p1.PercentOfClass != 8.5 {
		t.Errorf("Expected person 1 percent 8.5, got %.1f", p1.PercentOfClass)
	}
	if p1.SoleVotingPower != 0 {
		t.Errorf("Expected person 1 sole voting 0, got %d", p1.SoleVotingPower)
	}
	if p1.SharedVotingPower != 2100000 {
		t.Errorf("Expected person 1 shared voting 2100000, got %d", p1.SharedVotingPower)
	}
	if p1.TypeOfReportingPerson != "PN" {
		t.Errorf("Expected person 1 type 'PN', got '%s'", p1.TypeOfReportingPerson)
	}

	// Second reporting person: Leonard Braden Michael
	p2 := filing.ReportingPersons[1]
	if p2.CIK != "0001373603" {
		t.Errorf("Expected person 2 CIK '0001373603', got '%s'", p2.CIK)
	}
	if p2.Name != "Leonard Braden Michael" {
		t.Errorf("Expected person 2 name 'Leonard Braden Michael', got '%s'", p2.Name)
	}
	if p2.AggregateAmountOwned != 2435000 {
		t.Errorf("Expected person 2 aggregate 2435000, got %d", p2.AggregateAmountOwned)
	}
	if p2.PercentOfClass != 9.9 {
		t.Errorf("Expected person 2 percent 9.9, got %.1f", p2.PercentOfClass)
	}
	if p2.SoleVotingPower != 335000 {
		t.Errorf("Expected person 2 sole voting 335000, got %d", p2.SoleVotingPower)
	}
	if p2.SharedVotingPower != 2100000 {
		t.Errorf("Expected person 2 shared voting 2100000, got %d", p2.SharedVotingPower)
	}
	if p2.TypeOfReportingPerson != "IN" {
		t.Errorf("Expected person 2 type 'IN', got '%s'", p2.TypeOfReportingPerson)
	}

	// Test Items
	if filing.Items13D == nil {
		t.Fatal("Expected Items13D to be populated")
	}

	items := filing.Items13D

	// Item 4 is the most important - check it contains activist language
	if items.Item4PurposeOfTransaction == "" {
		t.Error("Expected Item4PurposeOfTransaction to be populated")
	}
	t.Logf("Item 4 (Purpose): %s", items.Item4PurposeOfTransaction[:100]+"...")

	// Item 3 - source of funds
	if items.Item3SourceOfFunds == "" {
		t.Error("Expected Item3SourceOfFunds to be populated")
	}

	// Test helper methods
	if filing.IsActivist() != true {
		t.Error("Expected IsActivist() to return true for 13D")
	}

	if filing.IsPassive() != false {
		t.Error("Expected IsPassive() to return false for 13D")
	}

	// Test total calculation (max percent since these are related persons)
	totalPercent := filing.CalculateTotalPercent()
	if totalPercent != 9.9 {
		t.Errorf("Expected total percent 9.9, got %.1f", totalPercent)
	}
}

func TestExtractAmendmentInfo(t *testing.T) {
	tests := []struct {
		formType      string
		wantAmendment bool
		wantNumber    *int
	}{
		{"SC 13D", false, nil},
		{"SCHEDULE 13D", false, nil},
		{"SC 13D/A", true, nil},
		{"SC 13D/A 2", true, ptrInt(2)},
		{"SC 13D/A#3", true, ptrInt(3)},
		{"SCHEDULE 13D/A Amendment No. 5", true, ptrInt(5)},
		{"SC 13G", false, nil},
		{"SC 13G/A", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.formType, func(t *testing.T) {
			gotAmendment, gotNumber := ExtractAmendmentInfo(tt.formType)

			if gotAmendment != tt.wantAmendment {
				t.Errorf("IsAmendment: got %v, want %v", gotAmendment, tt.wantAmendment)
			}

			if (gotNumber == nil) != (tt.wantNumber == nil) {
				t.Errorf("AmendmentNumber: got %v, want %v", gotNumber, tt.wantNumber)
			} else if gotNumber != nil && tt.wantNumber != nil && *gotNumber != *tt.wantNumber {
				t.Errorf("AmendmentNumber: got %d, want %d", *gotNumber, *tt.wantNumber)
			}
		})
	}
}

func ptrInt(i int) *int {
	return &i
}

func TestParseSchedule13G(t *testing.T) {
	// Test with edgartools reference data
	data, err := os.ReadFile("/home/nick/projects/port-edgartools/edgartools/tests/data/beneficial_ownership/schedule13g.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	filing, err := ParseSchedule13G(data)
	if err != nil {
		t.Fatalf("Failed to parse Schedule 13G: %v", err)
	}

	// Validate basic fields
	if filing.FormType != "SCHEDULE 13G" {
		t.Errorf("Expected FormType 'SCHEDULE 13G', got '%s'", filing.FormType)
	}

	if filing.IssuerCIK != "0001909747" {
		t.Errorf("Expected IssuerCIK '0001909747', got '%s'", filing.IssuerCIK)
	}

	if filing.IssuerName != "Jushi Holdings Inc." {
		t.Errorf("Expected IssuerName 'Jushi Holdings Inc.', got '%s'", filing.IssuerName)
	}

	if filing.IssuerCUSIP != "48213Y107" {
		t.Errorf("Expected IssuerCUSIP '48213Y107', got '%s'", filing.IssuerCUSIP)
	}

	if filing.SecurityTitle != "Subordinate Voting Shares, no par value" {
		t.Errorf("Expected SecurityTitle 'Subordinate Voting Shares, no par value', got '%s'", filing.SecurityTitle)
	}

	if filing.EventDate != "11/19/2025" {
		t.Errorf("Expected EventDate '11/19/2025', got '%s'", filing.EventDate)
	}

	// Validate rule designations
	if len(filing.RuleDesignations) == 0 {
		t.Error("Expected RuleDesignations to be populated")
	} else if filing.RuleDesignations[0] != "Rule 13d-1(c)" {
		t.Errorf("Expected RuleDesignations[0] 'Rule 13d-1(c)', got '%s'", filing.RuleDesignations[0])
	}

	// Validate reporting persons (should have 2 joint filers)
	if len(filing.ReportingPersons) != 2 {
		t.Fatalf("Expected 2 reporting persons, got %d", len(filing.ReportingPersons))
	}

	// First reporting person: Marex Securities Products Inc.
	p1 := filing.ReportingPersons[0]
	if p1.Name != "Marex Securities Products Inc." {
		t.Errorf("Expected person 1 name 'Marex Securities Products Inc.', got '%s'", p1.Name)
	}
	// Note: 13G often doesn't have CIK in person details, should fall back to filer CIK
	if p1.CIK != filing.FilerCIK {
		t.Errorf("Expected person 1 CIK to match filer CIK '%s', got '%s'", filing.FilerCIK, p1.CIK)
	}
	if p1.AggregateAmountOwned != 10000000 {
		t.Errorf("Expected person 1 aggregate 10000000, got %d", p1.AggregateAmountOwned)
	}
	if p1.PercentOfClass != 5.1 {
		t.Errorf("Expected person 1 percent 5.1, got %.1f", p1.PercentOfClass)
	}
	if p1.SoleVotingPower != 10000000 {
		t.Errorf("Expected person 1 sole voting 10000000, got %d", p1.SoleVotingPower)
	}
	if p1.SharedVotingPower != 0 {
		t.Errorf("Expected person 1 shared voting 0, got %d", p1.SharedVotingPower)
	}
	if p1.TypeOfReportingPerson != "CO" {
		t.Errorf("Expected person 1 type 'CO', got '%s'", p1.TypeOfReportingPerson)
	}
	if p1.MemberOfGroup != "a" {
		t.Errorf("Expected person 1 memberOfGroup 'a', got '%s'", p1.MemberOfGroup)
	}

	// Second reporting person: Marex Group plc
	p2 := filing.ReportingPersons[1]
	if p2.Name != "Marex Group plc" {
		t.Errorf("Expected person 2 name 'Marex Group plc', got '%s'", p2.Name)
	}
	if p2.AggregateAmountOwned != 10000000 {
		t.Errorf("Expected person 2 aggregate 10000000, got %d", p2.AggregateAmountOwned)
	}
	if p2.PercentOfClass != 5.1 {
		t.Errorf("Expected person 2 percent 5.1, got %.1f", p2.PercentOfClass)
	}
	if p2.MemberOfGroup != "a" {
		t.Errorf("Expected person 2 memberOfGroup 'a', got '%s'", p2.MemberOfGroup)
	}

	// Test joint filer aggregation
	// Both persons are memberOfGroup="a", so they're joint filers
	// Total should be MAX(10M, 10M) = 10M, NOT 20M!
	totalShares := filing.CalculateTotalShares()
	if totalShares != 10000000 {
		t.Errorf("Expected total shares 10000000 (joint filers), got %d", totalShares)
	}

	// Test Items
	if filing.Items13G == nil {
		t.Fatal("Expected Items13G to be populated")
	}

	items := filing.Items13G

	// Item 10 is important for 13G - passive certification
	if items.Item10Certification == "" {
		t.Error("Expected Item10Certification to be populated")
	}
	t.Logf("Item 10 (Certification): %s", items.Item10Certification[:min(100, len(items.Item10Certification))]+"...")

	// Item 3 should be N/A
	if !items.Item3NotApplicable {
		t.Error("Expected Item3NotApplicable to be true")
	}

	// Test helper methods
	if filing.IsActivist() != false {
		t.Error("Expected IsActivist() to return false for 13G")
	}

	if filing.IsPassive() != true {
		t.Error("Expected IsPassive() to return true for 13G")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
