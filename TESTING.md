# Testing Documentation for go-edgar

## Overview

go-edgar uses **data-driven tests** with JSON ground truth files. This makes it trivial to add new test cases without writing code.

## Test Structure

```
testdata/
└── form4/
    ├── README.md                    # How to add test cases
    ├── snow/                        # Test case: Snowflake
    │   ├── input.xml               # Form 4 XML to parse
    │   └── expected.json           # Expected output + metadata
    └── arrowhead_footnotes/         # Test case: Arrowhead
        ├── input.xml
        └── expected.json
```

## Test Coverage Summary

**Tests:** All passing ✅
- `TestForm4Parser` - Data-driven test (auto-discovers test cases)
- `TestTransactionCodeMapping` - Transaction code descriptions
- `TestJSONExport` - JSON serialization
- `TestInvalidXML` - Error handling
- `TestEmptyTransactionTable` - Edge case handling
- `TestDerivativeTransactions` - Derivative parsing, 10b5-1 detection, numeric conversions
- `TestValueNumericConversions` - Float64/Int methods with error handling
- `BenchmarkParse` - Performance benchmarking

## Current Test Cases

### 1. snow (Snowflake Inc.)
- **CIK:** 1640147
- **Focus:** Basic Form 4 parsing with CFO transaction
- **Tests:** Basic metadata, issuer info, reporting owner, transactions (non-derivative and derivative), helper methods

### 2. arrowhead_footnotes (Arrowhead Pharmaceuticals)
- **CIK:** 879407
- **Source:** https://www.sec.gov/Archives/edgar/data/879407/000124261525000006/wk-form4_1766530882.xml
- **Focus:** Edge case with multiple footnotes, 10b5-1 trading plan, weighted average prices
- **Tests:** Footnote parsing, footnote references, complex transaction scenarios

### 3. wave_derivatives (Wave Life Sciences)
- **CIK:** 1631574
- **Source:** https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml
- **Focus:** Derivative transactions (options, warrants), 10b5-1 plan with adoption date
- **Tests:** Derivative transaction parsing, exercise prices, underlying securities, 10b5-1 adoption date extraction

## Running Tests

```bash
# Run all tests
go test -v

# Run only the data-driven parser tests
go test -v -run TestForm4Parser

# Run specific test case
go test -v -run TestForm4Parser/snow
go test -v -run TestForm4Parser/arrowhead_footnotes

# Run with coverage
go test -cover

# Run benchmarks
go test -bench=.

# Run benchmarks with memory stats
go test -bench=BenchmarkParse -benchmem
```

## Test Output Example

```
=== RUN   TestForm4Parser
=== RUN   TestForm4Parser/arrowhead_footnotes
    form4_test.go:66: Source: https://www.sec.gov/Archives/edgar/data/879407/000124261525000006/wk-form4_1766530882.xml
    form4_test.go:67: Notes: Edge case: Multiple footnotes, 10b5-1 trading plan, weighted average prices, footnote references on shares/price/ownership fields
=== RUN   TestForm4Parser/snow
    form4_test.go:66: Source: https://www.sec.gov/cgi-bin/viewer?action=view&cik=1640147&accession_number=0001213900-22-069931&xbrl_type=v
    form4_test.go:67: Notes: Basic Form 4 test with Snowflake - validates parsing of issuer, reporting owner, address, and non-derivative transactions including exercise and sales
--- PASS: TestForm4Parser (0.00s)
    --- PASS: TestForm4Parser/arrowhead_footnotes (0.00s)
    --- PASS: TestForm4Parser/snow (0.00s)
PASS
ok      github.com/RxDataLab/go-edgar    0.005s
```

## Adding New Test Cases

### Golden File Workflow

The test framework uses **golden file testing** with auto-generation. Adding a new test case is simple:

1. **Create directory** in `testdata/form4/`:
   ```bash
   mkdir testdata/form4/my_new_case
   ```

2. **Add input.xml**:
   ```bash
   # Download from SEC, but be sure to add a header with email for SEC download
   curl "https://www.sec.gov/Archives/edgar/data/CIK/ACCESSION/form4.xml" \
     -o testdata/form4/my_new_case/input.xml
   ```

3. **Auto-generate expected.json** using the `-update` flag:
   ```bash
   go test -v -run TestForm4Parser/my_new_case -update
   ```

   This will:
   - Parse the input.xml
   - Generate a complete expected.json with all parsed fields
   - Save it to the test case directory

4. **Edit metadata** in the generated `expected.json`:
   ```json
   {
     "metadata": {
       "source_url": "https://www.sec.gov/Archives/...",
       "notes": "Description of what this test validates"
     },
     "expected": {
       // Auto-generated, don't edit this section
     }
   }
   ```

5. **Verify and commit**:
   ```bash
   # Run the test to verify it passes
   go test -v -run TestForm4Parser/my_new_case

   # Commit both files
   git add testdata/form4/my_new_case/
   git commit -m "Add my_new_case test"
   ```

### Regenerating Golden Files

If you update the parser and need to regenerate all expected.json files:

```bash
# Regenerate all test cases
go test -v -run TestForm4Parser -update

# Regenerate specific test case
go test -v -run TestForm4Parser/snow -update
```

**Important:** After regenerating, review the diffs to ensure changes are expected before committing.

### Ground Truth JSON Format

Each `expected.json` contains:

```json
{
  "metadata": {
    "source_url": "https://www.sec.gov/Archives/...",
    "notes": "What this test case validates (e.g., 'Edge case: Multiple reporting owners')"
  },
  "expected": {
    // Simplified Form4Output structure (table-like with numeric types)
    "formType": "4",
    "periodOfReport": "2025-12-19",
    "has10b51Plan": true,
    "issuer": {
      "cik": "0000879407",
      "name": "ARROWHEAD PHARMACEUTICALS, INC.",
      "ticker": "ARWR"
    },
    "transactions": [
      {
        "securityTitle": "Common Stock",
        "transactionDate": "2025-12-19",
        "transactionCode": "S",
        "shares": 50000,            // numeric, not string
        "pricePerShare": 13.20,     // numeric, not string
        "acquiredDisposed": "D",
        "sharesOwnedFollowing": 89218,
        "directIndirect": "D",
        "equitySwapInvolved": false,
        "is10b51Plan": true,        // per-transaction flag
        "footnotes": ["F1", "F4"]   // array of IDs
      }
    ]
    // ... complete simplified structure
  }
}
```

### File Naming Convention

**Directory names:** Use descriptive names like:
- `snow` - Company name
- `arrowhead_footnotes` - Company + feature
- `multiple_owners` - Feature being tested

**Files:**
- Always `input.xml` (the Form 4 XML)
- Always `expected.json` (expected parsed output + metadata)

## How Tests Work

The `TestForm4Parser` function:

1. **Discovers** all subdirectories in `testdata/form4/`
2. For each directory:
   - Loads `input.xml`
   - Loads `expected.json` (contains Form4Output structure)
   - Parses the XML to Form4 struct
   - **Converts** to Form4Output (simplified structure)
   - **Compares** actual vs expected using deep equality (`go-cmp`)
   - **Validates** helper methods on raw Form4 (GetMarketTrades, GetPurchases, GetSales)
3. Reports any mismatches with detailed diffs

### Comparison Method

Uses `github.com/google/go-cmp/cmp` for deep equality:
- Shows exact field-by-field diffs on failure
- Clear `-expected +actual` format
- Handles nested structs, arrays, pointers

## Test Categories

### Parsing Tests
- ✅ Basic Form 4 parsing (snow)
- ✅ Complex footnotes (arrowhead_footnotes)
- ✅ Invalid XML handling
- ✅ Empty transaction table handling

### Transaction Filtering Tests
- ✅ GetMarketTrades() - Filters P and S codes
- ✅ GetPurchases() - Only P codes
- ✅ GetSales() - Only S codes
- ✅ Helper method validation for all test cases

### Data Export Tests
- ✅ JSON marshaling/unmarshaling
- ✅ Round-trip JSON conversion

### Code Mapping Tests
- ✅ Transaction code descriptions (P, S, M, A, F, G, D)

## Performance Benchmarks

Current performance:
- **Parse time:** ~0.5ms per Form 4
- **Memory:** ~100KB per Form 4 object

Run benchmarks:
```bash
go test -bench=BenchmarkParse -benchmem
```

## Recommended Future Test Cases

High-value additions:
- [x] Form 4 with derivative transactions (options, warrants) - wave_derivatives
- [ ] Form 4 with multiple reporting owners
- [ ] Form 4 with indirect ownership
- [ ] Form 4 with grants/awards (A code)
- [ ] Form 4 with gifts (G code)
- [ ] Form 4 with very long footnotes (>1000 chars)
- [ ] Form 4 with special characters in owner names
- [ ] Form 4 with missing optional fields
- [ ] Form 4 with derivative holdings (no transactions, only holdings)
- [ ] Form 4 with equity swap involved flag

## Test Philosophy

1. **Real data over synthetic** - All test cases use real SEC filings
2. **Golden file testing** - Auto-generate comprehensive expected output, commit as regression baseline
3. **Structured ground truth** - JSON is unambiguous and machine-readable
4. **Easy to expand** - Adding tests requires no code changes, just `-update` flag
5. **Self-documenting** - Metadata describes what each test validates
6. **Automatic discovery** - No test registration needed
7. **Comprehensive coverage** - Test ALL parsed fields, not just a subset

## Debugging Failed Tests

When a test fails, the output shows:
```
parsed Form4 mismatch (-expected +actual):
  Form4{
    ...
-   DocumentType: "4",
+   DocumentType: "3",
    ...
  }
```

This makes it easy to:
1. Identify the exact field that differs
2. See expected vs actual values
3. Determine if it's a parsing bug or incorrect ground truth

## CI/CD Integration

Tests are designed for CI:
- Fast (<10ms total)
- No external dependencies
- Deterministic results
- Clear pass/fail output

Recommended GitHub Actions workflow:
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test -v -cover
```

## For AI Agents

When adding test cases:
1. Create directory in `testdata/form4/` and add `input.xml`
2. Run `go test -v -run TestForm4Parser/case_name -update` to auto-generate `expected.json`
3. Edit metadata section: add meaningful `source_url` and `notes`
4. Run test again without `-update` to verify it passes
5. Name test case directories descriptively (company_feature or just feature)

The test framework handles:
- Auto-discovery of new test cases
- Deep equality comparison with detailed diffs
- Helper method validation (GetMarketTrades, GetPurchases, GetSales)
- Comprehensive field-by-field testing

**Golden file workflow advantages:**
- Tests ALL parsed fields automatically
- Easy maintenance (regenerate with `-update` when parser changes)
- No manual JSON editing (except metadata)
- Catches regressions in any field
