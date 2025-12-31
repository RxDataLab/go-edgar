# XBRL Parsing Documentation

## Overview

go-edgar supports parsing **inline XBRL (iXBRL)** documents from SEC 10-K and 10-Q filings. Modern filings (2019+) use inline XBRL, where XBRL facts are embedded directly in HTML documents.

## Quick Start

```go
// Parse any XBRL file (auto-detects inline vs standalone)
data, _ := os.ReadFile("filing.htm")
xbrl, _ := edgar.ParseXBRLAuto(data)

// Extract key metrics
cash, _ := xbrl.GetCashAndEquivalents()
burn, _ := xbrl.GetBurn("")
revenue, _ := xbrl.GetRevenue("")

fmt.Printf("Cash: $%.2fB, Burn: $%.2fB\n", cash/1e9, burn/1e9)
```

## Architecture

### Core Components

1. **`xbrl.go`** - Core data structures (XBRL, Fact, Context, Period, Unit)
2. **`xbrl_ixbrl.go`** - Inline XBRL parser for HTML-embedded facts
3. **`xbrl_concepts.go`** - Concept mapping system (XBRL → standardized labels)
4. **`xbrl_financials.go`** - Query interface and high-level helpers
5. **`concept_mappings.json`** - Embedded mapping definitions (compiled into binary)

### Data Flow

```
SEC Filing (HTML with iXBRL)
         ↓
  ParseInlineXBRL()
         ↓
  Extract contexts, units, facts
         ↓
  Apply concept mappings
         ↓
  Query interface (ByLabel, ByPeriod, etc.)
         ↓
  High-level helpers (GetCash, GetBurn, etc.)
         ↓
  Structured output (FinancialSnapshot)
```

## Inline XBRL Format

Modern 10-K/10-Q filings use inline XBRL with the following structure:

```html
<html xmlns:ix="http://www.xbrl.org/2013/inlineXBRL"
      xmlns:xbrli="http://www.xbrl.org/2003/instance"
      xmlns:us-gaap="http://fasb.org/us-gaap/2024">

  <body>
    <!-- Human-readable content with embedded facts -->
    Cash and equivalents:
    <ix:nonFraction contextRef="c-4"
                    name="us-gaap:CashAndCashEquivalentsAtCarryingValue"
                    unitRef="usd"
                    decimals="-6">
      1234000000
    </ix:nonFraction>

    <!-- Contexts and units defined in hidden section -->
    <div style="display:none">
      <ix:header>
        <ix:resources>
          <xbrli:context id="c-4">
            <xbrli:entity>
              <xbrli:identifier scheme="http://www.sec.gov/CIK">0001682852</xbrli:identifier>
            </xbrli:entity>
            <xbrli:period>
              <xbrli:instant>2024-12-31</xbrli:instant>
            </xbrli:period>
          </xbrli:context>

          <xbrli:unit id="usd">
            <xbrli:measure>iso4217:USD</xbrli:measure>
          </xbrli:unit>
        </ix:resources>
      </ix:header>
    </div>
  </body>
</html>
```

### Key Elements

| Element | Purpose | Example |
|---------|---------|---------|
| `ix:nonFraction` | Numeric facts | Cash, revenue, shares |
| `ix:nonNumeric` | Text facts | Company name, dates |
| `contextRef` | Links to period/dimensions | c-4, c-1, etc. |
| `unitRef` | Links to measurement unit | usd, shares, usdPerShare |
| `decimals` | Precision (-6 = millions) | -3, -6, 0, INF |

## Concept Mapping System

The parser maps US-GAAP concepts to standardized labels for easy querying.

### How It Works

**1. Mapping Definition** (`concept_mappings.json`):
```json
{
  "Cash and Cash Equivalents": {
    "concepts": [
      "us-gaap:CashAndCashEquivalentsAtCarryingValue",
      "us-gaap:Cash",
      "us-gaap:CashAndCashEquivalents"
    ],
    "notes": "Primary liquidity measure"
  }
}
```

**2. Embedded at Compile Time** (using `go:embed`):
- JSON file is embedded in the binary
- No external dependencies at runtime
- Fast lookup with in-memory hash maps

**3. Usage**:
```go
// Query by standardized label (not XBRL concept)
fact, _ := xbrl.Query().ByLabel("Cash and Cash Equivalents").MostRecent()
cash, _ := fact.Float64()
```

### Adding New Mappings

Simply edit `concept_mappings.json` and rebuild:

```json
{
  "Operating Cash Flow": {
    "concepts": [
      "us-gaap:NetCashProvidedByUsedInOperatingActivities"
    ],
    "notes": "Cash from operations"
  }
}
```

No code changes needed!

## Query Interface

The fluent query API allows flexible fact filtering:

```go
// Get most recent cash value
cash := xbrl.Query().
  ByLabel("Cash and Cash Equivalents").
  InstantOnly().
  MostRecent()

// Get R&D expense for specific period
rd := xbrl.Query().
  ByLabel("Research and Development Expense").
  DurationOnly().
  ForPeriodEndingOn("2024-12-31").
  First()

// Get all revenue facts
revenues := xbrl.Query().
  ByLabel("Revenue").
  Get()

// Sum all operating expenses
total := xbrl.Query().
  ByLabel("Total Operating Expenses").
  Sum()
```

### Filter Methods

| Method | Description | Example |
|--------|-------------|---------|
| `ByLabel(label)` | Filter by standardized label | "Cash and Cash Equivalents" |
| `ByConcept(concepts...)` | Filter by XBRL concept | "us-gaap:Cash" |
| `ForPeriodEndingOn(date)` | Filter by period end date | "2024-12-31" |
| `InstantOnly()` | Balance sheet items (point-in-time) | Cash, assets, liabilities |
| `DurationOnly()` | Income statement items (period) | Revenue, expenses |

### Retrieval Methods

| Method | Description | Returns |
|--------|-------------|---------|
| `Get()` | All matching facts | `[]Fact` |
| `First()` | First match | `*Fact, error` |
| `MostRecent()` | Latest by period end | `*Fact, error` |
| `Sum()` | Sum of all numeric facts | `float64, error` |

## High-Level Financial Helpers

For common biotech metrics, use convenience methods:

```go
// Balance sheet (instant)
cash, _ := xbrl.GetCashAndEquivalents()
debt, _ := xbrl.GetTotalDebt()

// Income statement (duration)
revenue, _ := xbrl.GetRevenue("")              // Most recent
rd, _ := xbrl.GetResearchAndDevelopment("")
ga, _ := xbrl.GetGeneralAndAdministrative("")
burn, _ := xbrl.GetBurn("")                    // R&D + G&A

// Share count
shares, _ := xbrl.GetDilutedShares("")

// All-in-one snapshot
snapshot, _ := xbrl.GetSnapshot()
fmt.Printf("Cash: $%.2fB, Burn: $%.2fB, Runway: %.1f quarters\n",
  snapshot.Cash/1e9, snapshot.Burn/1e9, snapshot.RunwayQuarters)
```

## Edge Cases & Limitations

### 1. **Multiple Contexts for Same Period**

**Issue**: Companies may report the same metric multiple times with different dimensional contexts (segments, geographies, etc.).

**Example**:
```xml
<!-- Total revenue -->
<ix:nonFraction contextRef="c-1" name="us-gaap:Revenues">1000000000</ix:nonFraction>

<!-- Revenue by segment -->
<ix:nonFraction contextRef="c-product-a" name="us-gaap:Revenues">600000000</ix:nonFraction>
<ix:nonFraction contextRef="c-product-b" name="us-gaap:Revenues">400000000</ix:nonFraction>
```

**Solution**: Our query filters by the **base context** (no dimensions). Dimensional data requires explicit context matching.

**Current Status**: ✅ Handled - We match contexts without dimensions for top-level metrics

### 2. **Decimal Scaling**

**Issue**: XBRL uses `decimals` attribute to indicate scale:
- `decimals="-6"` means value is in millions
- `decimals="-3"` means value is in thousands
- `decimals="0"` means exact value

**Example**:
```xml
<!-- Value is actually $1,234,000,000 (1.234B) -->
<ix:nonFraction decimals="-6">1234</ix:nonFraction>
```

**Solution**: `parseNumericValue()` automatically scales values.

**Current Status**: ✅ Implemented in `xbrl.go:parseNumericValue()`

### 3. **Fiscal Year vs Calendar Year**

**Issue**: Some companies have fiscal years that don't align with calendar years (e.g., FY ends June 30).

**Example**:
- Microsoft: FY2024 ends June 30, 2024
- Moderna: FY2024 ends December 31, 2024

**Solution**: Always check `Period.EndDate` explicitly instead of assuming calendar year.

**Current Status**: ⚠️ Partial - Tests use hardcoded dates, should validate fiscal year

**Recommendation**: Add fiscal year detection:
```go
func (x *XBRL) GetFiscalYearEnd() time.Time {
  // Find most recent instant context
}
```

### 4. **Company-Specific Extensions**

**Issue**: Companies define custom XBRL concepts not in US-GAAP taxonomy.

**Example**: Moderna uses `mrna:InventoryShelfLife`, which isn't a standard concept.

**Solution**: Only map US-GAAP concepts. Company extensions are unmapped.

**Current Status**: ✅ Working as designed - 18.8% mapping rate is expected

**Mapping Rate Analysis**:
- 744 total facts in Moderna 10-K
- 140 mapped to standardized labels (18.8%)
- Unmapped facts include:
  - Document metadata (`dei:*`)
  - Company extensions (`mrna:*`)
  - Extensible enumerations (positioning info)
  - Dimensional members

### 5. **Duplicate Facts with Different Precision**

**Issue**: Same fact can appear multiple times with different `decimals` values.

**Example**:
```xml
<ix:nonFraction decimals="-6">1234</ix:nonFraction>  <!-- Millions -->
<ix:nonFraction decimals="-3">1234567</ix:nonFraction>  <!-- Thousands -->
```

**Solution**: Currently, we capture all facts. Query methods like `MostRecent()` return the first match.

**Current Status**: ⚠️ Needs improvement

**Recommendation**: Prefer higher precision:
```go
// In MostRecent(), if multiple facts match:
// 1. Sort by precision (higher decimals = less scale = more precision)
// 2. Return highest precision value
```

### 6. **Text vs Numeric Facts**

**Issue**: `ix:nonNumeric` contains text, `ix:nonFraction` contains numbers.

**Example**:
```xml
<ix:nonNumeric name="dei:EntityRegistrantName">Moderna, Inc.</ix:nonNumeric>
<ix:nonFraction name="us-gaap:Cash">1234000000</ix:nonFraction>
```

**Solution**: Both are captured as `Fact.Value` (string). Use `Fact.Float64()` to convert numeric facts.

**Current Status**: ✅ Working - `Float64()` returns error for non-numeric values

### 7. **Missing Concepts**

**Issue**: Not all companies report all metrics. Pre-revenue biotechs may have $0 revenue, no debt, etc.

**Example**: Early-stage biotech has no revenue, no debt.

**Solution**: Helper methods return errors when facts aren't found. Handle gracefully:

```go
revenue, err := xbrl.GetRevenue("")
if err != nil {
  revenue = 0  // Pre-revenue company
}
```

**Current Status**: ✅ Implemented - All helpers return `error`

### 8. **Character Encoding**

**Issue**: SEC filings may declare encoding as "ASCII" but contain UTF-8 characters.

**Solution**: Set `CharsetReader` to treat all encodings as UTF-8.

**Current Status**: ✅ Fixed in `xbrl_ixbrl.go:37-40`

```go
decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
  return input, nil  // Treat all as UTF-8
}
```

## Testing

### Test Data

- **Moderna 10-K (FY2024)** - Real inline XBRL filing
  - File: `testdata/xbrl/moderna_10k/input.htm` (2.6 MB)
  - Contexts: 368
  - Units: 15
  - Facts: 744
  - Mapped concepts: 140 (18.8%)

### Test Coverage

| Test | Purpose | File |
|------|---------|------|
| `TestParseInlineXBRL_Moderna` | End-to-end parsing and extraction | `xbrl_test.go:12` |
| `TestDetectXBRLType` | Format auto-detection | `xbrl_test.go:229` |
| `TestXBRLFactExtraction` | Concept mapping validation | `xbrl_test.go:253` |
| `TestConceptMappings` | Mapping system | `xbrl_concepts_test.go:7` |

### Expected Output

```
=== Moderna FY2024 Financial Snapshot ===
Period: 2024-12-31

Metric                         Value
------                         -----
Cash & Equivalents            $1.93B
Total Debt                        $0
Revenue                       $3.24B
R&D Expense                   $4.54B
G&A Expense                   $1.17B
Burn Rate (R&D + G&A)         $5.72B
Diluted Shares                384.0M
Runway (Cash/Quarterly Burn)     1.3 quarters
```

### Adding Test Cases

To add a new biotech company:

1. **Download 10-K**:
```go
func TestDownloadBiotech10K(t *testing.T) {
  // See download_10k_test.go for template
}
```

2. **Create directory**: `testdata/xbrl/{company}_10k/`

3. **Add metadata**: `metadata.json`

4. **Run test**: Verify extraction works

## Performance

Tested on Moderna 10-K (2.6 MB HTML):

- **Parse time**: ~70ms (contexts, units, facts)
- **Memory**: ~5 MB (in-memory structures)
- **Facts extracted**: 744
- **Throughput**: ~35 MB/s

## Future Enhancements

### Priority 1: Production Readiness
- [ ] Handle duplicate facts (prefer higher precision)
- [ ] Add fiscal year detection
- [ ] Support for 10-Q (quarterly) filings
- [ ] Validate fact values (sanity checks)

### Priority 2: Extended Coverage
- [ ] Add more concept mappings (operating cash flow, capex, etc.)
- [ ] Support Form 13F (holdings)
- [ ] Support Form 8-K (current events)

### Priority 3: Advanced Features
- [ ] Dimensional data support (revenue by segment)
- [ ] Time series extraction (multi-period queries)
- [ ] CSV export
- [ ] Footnote resolution

## Troubleshooting

### "failed to extract resources: xml: encoding 'ASCII' declared..."

**Solution**: Upgrade to latest version - CharsetReader is set to handle all encodings.

### "no facts found"

**Possible causes**:
1. Wrong XBRL format (standalone vs inline) - use `ParseXBRLAuto()`
2. File is not XBRL - check `DetectXBRLType()`
3. Malformed XML - check file integrity

### "cash and equivalents not found"

**Possible causes**:
1. Company uses different concept - check raw facts with `xbrl.Facts`
2. Wrong period - specify exact date with `.ForPeriodEndingOn("2024-12-31")`
3. Not yet mapped - add to `concept_mappings.json`

### Low mapping rate (<10%)

**Expected**: Mapping rate varies by company. 15-25% is typical for biotech.

**To improve**: Add company-specific concepts to mappings or query by raw concept name.

## References

- **XBRL Spec**: https://www.xbrl.org/Specification/inlineXBRL-part1/REC-2013-11-18/inlineXBRL-part1-REC-2013-11-18.html
- **US-GAAP Taxonomy**: https://xbrl.us/home/filers/sec-reporting/taxonomies/
- **SEC EDGAR**: https://www.sec.gov/edgar/searchedgar/accessing-edgar-data.htm

## License

Part of go-edgar - see main README for license info.
