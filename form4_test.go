package edgar_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/RxDataLab/go-edgar"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update golden test files")

// TestCaseMetadata contains metadata about a test case
type TestCaseMetadata struct {
	SourceURL string `json:"source_url"`
	Notes     string `json:"notes"`
}

// Form4TestCase represents a complete test case with metadata and expected output
type Form4TestCase struct {
	Metadata TestCaseMetadata   `json:"metadata"`
	Expected *edgar.Form4Output `json:"expected"`
}

// TestForm4Parser is a data-driven test that discovers and tests all Form 4 test cases
// Test cases are stored in testdata/form4/<case_name>/ with:
//   - input.xml: The Form 4 XML file
//   - expected.json: The expected parsed output with metadata
func TestForm4Parser(t *testing.T) {
	testCasesDir := "testdata/form4"

	// Discover all test case directories
	entries, err := os.ReadDir(testCasesDir)
	require.NoError(t, err, "failed to read test cases directory")

	var testCases []string
	for _, entry := range entries {
		if entry.IsDir() {
			testCases = append(testCases, entry.Name())
		}
	}

	require.NotEmpty(t, testCases, "no test cases found in %s", testCasesDir)

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			casePath := filepath.Join(testCasesDir, testCase)
			inputPath := filepath.Join(casePath, "input.xml")
			expectedPath := filepath.Join(casePath, "expected.json")

			// Load input XML
			xmlData, err := os.ReadFile(inputPath)
			require.NoError(t, err, "failed to read input.xml")

			// Load expected output
			expectedData, err := os.ReadFile(expectedPath)
			require.NoError(t, err, "failed to read expected.json")

			var tc Form4TestCase
			err = json.Unmarshal(expectedData, &tc)
			require.NoError(t, err, "failed to parse expected.json")

			// Log metadata
			t.Logf("Source: %s", tc.Metadata.SourceURL)
			t.Logf("Notes: %s", tc.Metadata.Notes)

			// Parse the Form 4 (raw XML -> Form4 struct)
			form4, err := edgar.Parse(xmlData)
			require.NoError(t, err, "failed to parse Form 4")

			// Convert to output format (simplified structure)
			// This is what the CLI actually outputs
			freshOutput := form4.ToOutput()

			// ALWAYS compare fresh output with committed golden file
			// This ensures golden files stay up to date with parser changes
			if diff := cmp.Diff(tc.Expected, freshOutput); diff != "" {
				// Write the fresh output to a .new file for review
				newPath := expectedPath + ".new"
				tc.Expected = freshOutput
				newData, err := json.MarshalIndent(tc, "", "  ")
				require.NoError(t, err, "failed to marshal new output")

				err = os.WriteFile(newPath, newData, 0o644)
				require.NoError(t, err, "failed to write .new file")

				if *updateGolden {
					// -update flag: Copy .new to actual golden file
					err = os.WriteFile(expectedPath, newData, 0o644)
					require.NoError(t, err, "failed to update golden file")

					// Remove .new file after accepting
					os.Remove(newPath)

					t.Logf("✓ Accepted new snapshot: %s", expectedPath)
				} else {
					// No -update flag: Fail with helpful message and show diff
					t.Errorf("Snapshot mismatch!\n\n"+
						"DIFF (-committed +fresh):\n%s\n\n"+
						"A new snapshot has been written to:\n  %s\n\n"+
						"To review the change:\n"+
						"  diff %s %s\n\n"+
						"If the new output is CORRECT, accept it with:\n"+
						"  go test -v -run TestForm4Parser/%s -update\n\n"+
						"If the new output is WRONG, fix the parser and re-run tests.\n"+
						"The .new file will be automatically cleaned up on next test run.",
						diff, newPath, expectedPath, newPath, testCase)
				}
			} else {
				// Output matches golden file - clean up any stale .new files
				newPath := expectedPath + ".new"
				if _, err := os.Stat(newPath); err == nil {
					os.Remove(newPath)
				}
			}

			// Additional verification: test helper methods on the raw Form4 struct
			verifyHelperMethods(t, form4)
		})
	}
}

// verifyHelperMethods tests the GetMarketTrades, GetPurchases, GetSales methods
func verifyHelperMethods(t *testing.T, f4 *edgar.Form4) {
	marketTrades := f4.GetMarketTrades()
	purchases := f4.GetPurchases()
	sales := f4.GetSales()

	// Verify all market trades are P or S
	for _, trade := range marketTrades {
		assert.Contains(t, []string{"P", "S"}, trade.Coding.Code,
			"market trade should have P or S code")
	}

	// Verify all purchases are P
	for _, p := range purchases {
		assert.Equal(t, "P", p.Coding.Code, "purchase should have P code")
	}

	// Verify all sales are S
	for _, s := range sales {
		assert.Equal(t, "S", s.Coding.Code, "sale should have S code")
	}

	// Verify purchases + sales = market trades
	assert.Equal(t, len(purchases)+len(sales), len(marketTrades),
		"purchases + sales should equal market trades")
}

// TestTransactionCodeMapping verifies transaction code descriptions
func TestTransactionCodeMapping(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"P", "Open Market Purchase"},
		{"S", "Open Market Sale"},
		{"M", "Exercise or Conversion of Derivative Security"},
		{"A", "Grant, Award or Other Acquisition"},
		{"F", "Payment of Exercise Price or Tax Liability"},
		{"G", "Gift"},
		{"D", "Disposition to the Issuer"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			desc := edgar.TransactionCodeDescription(tt.code)
			assert.Equal(t, tt.expected, desc)
		})
	}
}

// TestJSONExport verifies Form4 can be marshaled and unmarshaled to JSON
func TestJSONExport(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/form4/snow/input.xml")
	require.NoError(t, err)

	f4, err := edgar.Parse(xmlData)
	require.NoError(t, err)

	// Export to JSON
	jsonData, err := json.MarshalIndent(f4, "", "  ")
	require.NoError(t, err)

	// Verify we can unmarshal it back
	var f4Copy edgar.Form4
	err = json.Unmarshal(jsonData, &f4Copy)
	require.NoError(t, err)

	assert.Equal(t, f4.DocumentType, f4Copy.DocumentType)
	assert.Equal(t, f4.Issuer.Name, f4Copy.Issuer.Name)
}

// TestInvalidXML verifies error handling for invalid XML
func TestInvalidXML(t *testing.T) {
	invalidXML := []byte(`<invalid>not a form 4</invalid>`)

	_, err := edgar.Parse(invalidXML)
	assert.Error(t, err, "should fail on invalid XML")
}

// TestEmptyTransactionTable verifies handling of Form 4 with no transactions
func TestEmptyTransactionTable(t *testing.T) {
	minimalXML := []byte(`
		<ownershipDocument>
			<documentType>4</documentType>
			<periodOfReport>2024-01-01</periodOfReport>
			<issuer>
				<issuerCik>1234567</issuerCik>
				<issuerName>Test Company</issuerName>
				<issuerTradingSymbol>TEST</issuerTradingSymbol>
			</issuer>
			<reportingOwner>
				<reportingOwnerId>
					<rptOwnerCik>7654321</rptOwnerCik>
					<rptOwnerName>Test Owner</rptOwnerName>
				</reportingOwnerId>
				<reportingOwnerRelationship>
					<isDirector>1</isDirector>
					<isOfficer>0</isOfficer>
				</reportingOwnerRelationship>
			</reportingOwner>
		</ownershipDocument>
	`)

	f4, err := edgar.Parse(minimalXML)
	require.NoError(t, err)

	assert.Equal(t, "4", f4.DocumentType)
	assert.Equal(t, "Test Company", f4.Issuer.Name)

	// Should handle nil table gracefully
	trades := f4.GetMarketTrades()
	assert.Empty(t, trades)
}

// BenchmarkParse benchmarks Form 4 parsing performance
func BenchmarkParse(b *testing.B) {
	xmlData, err := os.ReadFile("testdata/form4/snow/input.xml")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = edgar.Parse(xmlData)
	}
}

// TestDerivativeTransactions tests derivative-specific functionality
func TestDerivativeTransactions(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/form4/wave_derivatives/input.xml")
	require.NoError(t, err)

	f4, err := edgar.Parse(xmlData)
	require.NoError(t, err)

	// Test derivative parsing
	require.NotNil(t, f4.DerivativeTable)
	require.NotEmpty(t, f4.DerivativeTable.Transactions)

	// Test 10b5-1 detection
	assert.True(t, f4.Is10b51Plan(), "should detect 10b5-1 plan")

	adoptionDate := f4.Get10b51AdoptionDate()
	assert.Equal(t, "March 13, 2025", adoptionDate, "should extract adoption date")

	// Test derivative structure
	firstDeriv := f4.DerivativeTable.Transactions[0]
	assert.Equal(t, "Share Option (right to buy)", firstDeriv.SecurityTitle)

	// Test numeric conversions
	price, err := firstDeriv.ConversionOrExercisePrice.Float64()
	require.NoError(t, err)
	assert.Equal(t, 2.83, price)

	// Test underlying security
	assert.Equal(t, "Ordinary Shares", firstDeriv.UnderlyingSecurity.SecurityTitle.Value)
	shares, err := firstDeriv.UnderlyingSecurity.Shares.Int()
	require.NoError(t, err)
	assert.Equal(t, 60000, shares)

	// Test exerciseDate footnote reference (should have footnote, empty value)
	assert.Equal(t, "", firstDeriv.ExerciseDate.Value)
	assert.Equal(t, "F2", firstDeriv.ExerciseDate.FootnoteID.ID)

	// Test transaction under 10b5-1
	nonDerivTxn := f4.NonDerivativeTable.Transactions[0]
	assert.True(t, nonDerivTxn.IsUnder10b51(f4), "transaction should be under 10b5-1 plan")
}

// TestValueNumericConversions tests Float64 and Int methods
func TestValueNumericConversions(t *testing.T) {
	tests := []struct {
		name        string
		value       edgar.Value
		expectFloat float64
		expectInt   int
		shouldError bool
	}{
		{
			name:        "valid float",
			value:       edgar.Value{Value: "123.45"},
			expectFloat: 123.45,
			shouldError: false,
		},
		{
			name:        "valid int",
			value:       edgar.Value{Value: "60000"},
			expectFloat: 60000.0,
			expectInt:   60000,
			shouldError: false,
		},
		{
			name:        "empty value",
			value:       edgar.Value{Value: ""},
			shouldError: true,
		},
		{
			name: "footnote reference only",
			value: edgar.Value{
				Value:      "",
				FootnoteID: edgar.FootnoteID{ID: "F1"},
			},
			shouldError: true,
		},
		{
			name: "value with footnote",
			value: edgar.Value{
				Value:      "2.83",
				FootnoteID: edgar.FootnoteID{ID: "F1"},
			},
			expectFloat: 2.83,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Float64
			f, err := tt.value.Float64()
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, tt.expectFloat, f, 0.001)
			}

			// Test Int (only if no decimal)
			if tt.expectInt > 0 {
				i, err := tt.value.Int()
				require.NoError(t, err)
				assert.Equal(t, tt.expectInt, i)
			}
		})
	}
}
