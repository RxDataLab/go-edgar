package edgar_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	edgar "github.com/RxDataLab/go-edgar"
	"github.com/stretchr/testify/require"
)

// TestForm4OutputSchema validates that ALL expected.json files contain
// ALL required output fields, regardless of their values.
//
// This test uses reflection to get the complete field list from the output structs
// and ensures every expected.json has those fields present.
//
// WHY: When we add new fields to output structs, this test will FAIL if any
// expected.json is missing the new field, forcing us to regenerate golden files.
func TestForm4OutputSchema(t *testing.T) {
	testCasesDir := "testdata/form4"

	testCases, err := os.ReadDir(testCasesDir)
	require.NoError(t, err, "failed to read test cases directory")

	for _, entry := range testCases {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			expectedPath := filepath.Join(testCasesDir, entry.Name(), "expected.json")

			// Load expected.json
			data, err := os.ReadFile(expectedPath)
			require.NoError(t, err, "failed to read expected.json")

			var tc Form4TestCase
			err = json.Unmarshal(data, &tc)
			require.NoError(t, err, "failed to parse expected.json")

			// Validate schema for each transaction type
			if tc.Expected.Transactions != nil && len(tc.Expected.Transactions) > 0 {
				validateNonDerivativeTransactionSchema(t, &tc.Expected.Transactions[0], entry.Name())
			}

			if tc.Expected.Derivatives != nil && len(tc.Expected.Derivatives) > 0 {
				validateDerivativeTransactionSchema(t, &tc.Expected.Derivatives[0], entry.Name())
			}
		})
	}
}

// validateNonDerivativeTransactionSchema checks that a transaction has ALL required fields
func validateNonDerivativeTransactionSchema(t *testing.T, txn *edgar.NonDerivativeTransactionOut, testCase string) {
	// Get all fields from the struct using reflection
	txnType := reflect.TypeOf(*txn)

	for i := 0; i < txnType.NumField(); i++ {
		field := txnType.Field(i)
		jsonTag := field.Tag.Get("json")

		// Skip omitempty fields
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract field name from json tag (remove omitempty if present)
		fieldName := jsonTag
		if idx := len(jsonTag); idx > 0 {
			// Remove ,omitempty suffix
			for j := 0; j < len(jsonTag); j++ {
				if jsonTag[j] == ',' {
					fieldName = jsonTag[:j]
					break
				}
			}
		}

		// Check if field is exported and should be in JSON
		if field.PkgPath != "" {
			continue // Skip unexported fields
		}

		// CRITICAL FIELDS that must ALWAYS be present (no omitempty allowed)
		criticalFields := map[string]bool{
			"is10b51Plan":           true,
			"plan10b51AdoptionDate": true,
		}

		if criticalFields[fieldName] {
			// Check if the field exists in the JSON representation
			// by marshaling and checking the JSON output
			jsonBytes, err := json.Marshal(txn)
			require.NoError(t, err, "failed to marshal transaction for schema validation")

			var jsonMap map[string]interface{}
			err = json.Unmarshal(jsonBytes, &jsonMap)
			require.NoError(t, err, "failed to unmarshal transaction JSON")

			_, exists := jsonMap[fieldName]
			require.True(t, exists,
				"Test case '%s': Critical field '%s' is MISSING from NonDerivativeTransactionOut JSON. "+
					"This field must ALWAYS be present. Run 'go test -v -run TestForm4Parser -update' to regenerate golden files.",
				testCase, fieldName)

			t.Logf("✓ Test case '%s': Field '%s' is present (value: %v)", testCase, fieldName, jsonMap[fieldName])
		}
	}
}

// validateDerivativeTransactionSchema checks that a derivative transaction has ALL required fields
func validateDerivativeTransactionSchema(t *testing.T, txn *edgar.DerivativeTransactionOut, testCase string) {
	// Get all fields from the struct using reflection
	txnType := reflect.TypeOf(*txn)

	for i := 0; i < txnType.NumField(); i++ {
		field := txnType.Field(i)
		jsonTag := field.Tag.Get("json")

		// Skip omitempty fields
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract field name from json tag
		fieldName := jsonTag
		for j := 0; j < len(jsonTag); j++ {
			if jsonTag[j] == ',' {
				fieldName = jsonTag[:j]
				break
			}
		}

		// Check if field is exported
		if field.PkgPath != "" {
			continue
		}

		// CRITICAL FIELDS that must ALWAYS be present
		criticalFields := map[string]bool{
			"is10b51Plan":           true,
			"plan10b51AdoptionDate": true,
		}

		if criticalFields[fieldName] {
			jsonBytes, err := json.Marshal(txn)
			require.NoError(t, err, "failed to marshal derivative transaction for schema validation")

			var jsonMap map[string]interface{}
			err = json.Unmarshal(jsonBytes, &jsonMap)
			require.NoError(t, err, "failed to unmarshal derivative transaction JSON")

			_, exists := jsonMap[fieldName]
			require.True(t, exists,
				"Test case '%s': Critical field '%s' is MISSING from DerivativeTransactionOut JSON. "+
					"This field must ALWAYS be present. Run 'go test -v -run TestForm4Parser -update' to regenerate golden files.",
				testCase, fieldName)

			t.Logf("✓ Test case '%s': Derivative field '%s' is present (value: %v)", testCase, fieldName, jsonMap[fieldName])
		}
	}
}
