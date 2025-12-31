package edgar

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestParseInlineXBRL_Moderna(t *testing.T) {
	// Load the Moderna 10-K
	data, err := os.ReadFile("testdata/xbrl/moderna_10k/input.htm")
	if err != nil {
		t.Fatalf("Failed to read Moderna 10-K: %v", err)
	}

	t.Logf("Loaded %d bytes from Moderna 10-K", len(data))

	// Detect XBRL type
	xbrlType := DetectXBRLType(data)
	if xbrlType != "inline" {
		t.Fatalf("Expected inline XBRL, got %s", xbrlType)
	}
	t.Logf("✓ Detected inline XBRL format")

	// Parse the iXBRL
	xbrl, err := ParseInlineXBRL(data)
	if err != nil {
		t.Fatalf("Failed to parse iXBRL: %v", err)
	}

	// Verify we extracted data
	t.Logf("✓ Parsed XBRL successfully")
	t.Logf("  - Contexts: %d", len(xbrl.Contexts))
	t.Logf("  - Units: %d", len(xbrl.Units))
	t.Logf("  - Facts: %d", len(xbrl.Facts))

	if len(xbrl.Contexts) == 0 {
		t.Error("No contexts extracted")
	}
	if len(xbrl.Units) == 0 {
		t.Error("No units extracted")
	}
	if len(xbrl.Facts) == 0 {
		t.Fatal("No facts extracted")
	}

	// Print sample facts
	t.Logf("\nSample facts (first 10):")
	for i := 0; i < 10 && i < len(xbrl.Facts); i++ {
		fact := xbrl.Facts[i]
		label := fact.StandardLabel
		if label == "" {
			label = "(unmapped)"
		}
		t.Logf("  [%d] %s = %s (context: %s, label: %s)",
			i+1, fact.Concept, fact.Value, fact.ContextRef, label)
	}

	// Extract minimal dataset
	t.Log("\n=== Extracting Key Financial Metrics ===")

	// Find the most recent fiscal year end context
	var fy2024Context string
	var fy2024Instant string
	for _, ctx := range xbrl.Contexts {
		if ctx.Period.EndDate == "2024-12-31" && ctx.Period.StartDate == "2024-01-01" {
			fy2024Context = ctx.ID
			t.Logf("✓ Found FY2024 duration context: %s", ctx.ID)
		}
		if ctx.Period.Instant == "2024-12-31" {
			fy2024Instant = ctx.ID
			t.Logf("✓ Found FY2024 instant context: %s", ctx.ID)
			break // Use first match
		}
	}

	if fy2024Context == "" {
		t.Error("Could not find FY2024 duration context")
	}
	if fy2024Instant == "" {
		t.Error("Could not find FY2024 instant context")
	}

	// Extract metrics using GetSnapshot()
	snapshot, err := xbrl.GetSnapshot()
	if err != nil {
		t.Fatalf("Failed to get snapshot: %v", err)
	}

	// Print results in table format
	t.Log("\n=== Moderna FY2024 Financial Snapshot ===")
	t.Logf("Company: %s", snapshot.CompanyName)
	t.Logf("Period: %s (%s)", snapshot.FiscalYearEnd, snapshot.FiscalPeriod)
	t.Logf("")

	// Print key metrics
	t.Logf("%-30s %20s", "Metric", "Value")
	t.Logf("%-30s %20s", "------", "-----")
	t.Logf("%-30s %20s", "Cash & Equivalents", formatCurrency(snapshot.Cash))
	t.Logf("%-30s %20s", "Total Assets", formatCurrency(snapshot.TotalAssets))
	t.Logf("%-30s %20s", "Total Liabilities", formatCurrency(snapshot.TotalLiabilities))
	t.Logf("%-30s %20s", "Stockholders Equity", formatCurrency(snapshot.StockholdersEquity))
	t.Logf("%-30s %20s", "Total Debt", formatCurrency(snapshot.TotalDebt))
	t.Logf("%-30s %20s", "Revenue", formatCurrency(snapshot.Revenue))
	t.Logf("%-30s %20s", "Net Income", formatCurrency(snapshot.NetIncome))
	t.Logf("%-30s %20s", "R&D Expense", formatCurrency(snapshot.RDExpense))
	t.Logf("%-30s %20s", "G&A Expense", formatCurrency(snapshot.GAExpense))
	t.Logf("%-30s %20s", "Cash Flow from Ops", formatCurrency(snapshot.CashFlowOperations))
	t.Logf("%-30s %20.1fM", "Diluted Shares", snapshot.DilutedShares/1_000_000)

	// Report missing required fields
	if len(snapshot.MissingRequiredFields) > 0 {
		t.Logf("\n⚠️  Missing Required Fields:")
		for _, field := range snapshot.MissingRequiredFields {
			t.Logf("  - %s", field)
		}
	}

	// Basic sanity checks
	if snapshot.Cash <= 0 {
		t.Errorf("Cash should be positive, got: %s", formatCurrency(snapshot.Cash))
	}
	if snapshot.RDExpense <= 0 {
		t.Errorf("R&D should be positive, got: %s", formatCurrency(snapshot.RDExpense))
	}
	if snapshot.TotalAssets <= 0 {
		t.Errorf("Total Assets should be positive, got: %s", formatCurrency(snapshot.TotalAssets))
	}

	t.Log("\n✅ All validations passed")
}

func formatCurrency(value float64) string {
	if value == 0 {
		return "$0"
	}

	billions := value / 1_000_000_000
	millions := value / 1_000_000

	if billions >= 1 {
		return fmt.Sprintf("$%.2fB", billions)
	}
	return fmt.Sprintf("$%.1fM", millions)
}

func TestDetectXBRLType(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "inline XBRL with xmlns",
			content:  `<html xmlns:ix="http://www.xbrl.org/2013/inlineXBRL">`,
			expected: "inline",
		},
		{
			name:     "inline XBRL with ix: tag",
			content:  `<ix:nonFraction contextRef="c1">123</ix:nonFraction>`,
			expected: "inline",
		},
		{
			name:     "standalone XBRL",
			content:  `<xbrl xmlns:xbrli="http://www.xbrl.org/2003/instance">`,
			expected: "standalone",
		},
		{
			name:     "unknown format",
			content:  `<html><body>No XBRL here</body></html>`,
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectXBRLType([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("DetectXBRLType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestXBRLFactExtraction tests that we can extract specific facts
func TestXBRLFactExtraction(t *testing.T) {
	// Load Moderna 10-K
	data, err := os.ReadFile("testdata/xbrl/moderna_10k/input.htm")
	if err != nil {
		t.Skip("Moderna 10-K not available for testing")
	}

	xbrl, err := ParseInlineXBRL(data)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Test concept mapping
	mappedCount := 0
	conceptCounts := make(map[string]int)

	for _, fact := range xbrl.Facts {
		if fact.StandardLabel != "" {
			mappedCount++
			conceptCounts[fact.StandardLabel]++
		}
	}

	t.Logf("Mapped %d/%d facts (%.1f%%)",
		mappedCount, len(xbrl.Facts),
		float64(mappedCount)/float64(len(xbrl.Facts))*100)

	// Print concept distribution
	t.Log("\nTop 20 standardized concepts:")
	type conceptCount struct {
		label string
		count int
	}
	var counts []conceptCount
	for label, count := range conceptCounts {
		counts = append(counts, conceptCount{label, count})
	}

	// Sort by count descending
	for i := 0; i < len(counts); i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[j].count > counts[i].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}

	for i := 0; i < 20 && i < len(counts); i++ {
		t.Logf("  %2d. %-40s %5d facts", i+1, counts[i].label, counts[i].count)
	}
}

// TestGenerateExpectedJSON generates expected output for the Moderna test case
// Run with: go test -v -run TestGenerateExpectedJSON
func TestGenerateExpectedJSON(t *testing.T) {
	t.Skip("Only run manually to generate expected output")

	// Load and parse Moderna 10-K
	data, err := os.ReadFile("testdata/xbrl/moderna_10k/input.htm")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	xbrl, err := ParseInlineXBRL(data)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	snapshot, err := xbrl.GetSnapshot()
	if err != nil {
		t.Fatalf("Failed to get snapshot: %v", err)
	}

	// Write to expected.json
	outFile, err := os.Create("testdata/xbrl/moderna_10k/expected.json")
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		t.Fatalf("Failed to encode JSON: %v", err)
	}

	t.Log("✅ Generated expected.json")
}
