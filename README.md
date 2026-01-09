# go-edgar

Fast, dependency-light Go parser for SEC filings. Used by [RxDataLab](https://rxdatalab.com/) for company analysis and in our data pipelines for [our biotech research application](https://app.rxdatalab.com/)

## Features

- **Minimal dependencies**
- **Multi-form support** - Form 4, Schedule 13D/G, 10-K/10-Q (XBRL)
- **SEC fetcher** - Built-in HTTP client with rate limiting
- **CLI tool** - Standalone binary for parsing and fetching filings
- **Fast** - Parse thousands of filings per second
- **Simple API** - Easy to use and understand
- **JSON export** - Clean, table-like output format

## Supported Forms

### Currently Supported

- ✅ **Form 4** - Insider trading filings (officers, directors, 10%+ owners)
  - Complete parsing of non-derivative AND derivative transactions
  - Automatic 10b5-1 trading plan detection with adoption dates
  - Transaction filtering (purchases, sales, market trades)
  - Footnote parsing and reference resolution

- ✅ **Schedule 13D/G** - 5%+ ownership filings (activist and passive investors)
  - Both XML and HTML format support
  - Amendment tracking and history
  - Item 4 parsing (activist intent)
  - Joint filer aggregation
  - Distinguishes between 13D (activist) and 13G (passive)

- ✅ **10-K/10-Q** - Annual and quarterly reports (XBRL/iXBRL)
  - Inline XBRL parser for modern SEC filings
  - 43 comprehensive GAAP concept mappings
  - Financial snapshot extraction (Cash, Revenue, R&D, G&A, Burn, etc.)
  - Balance sheet, income statement, cash flow, and per-share metrics

### Roadmap

- [ ] 13F - Institutional holdings
- [ ] Form D - Private placement offerings
- [ ] 8-K - Current events (with item type parsing)

## Installation

### As a Library

```bash
go get github.com/RxDataLab/go-edgar
```

### As a CLI Tool

```bash
# Build from source
git clone https://github.com/RxDataLab/go-edgar
cd go-edgar
make build

# Or use go install
go install github.com/RxDataLab/go-edgar/cmd/goedgar@latest
```

## CLI Usage

The `goedgar` CLI tool auto-detects form types and supports two modes:
1. **Single file mode** - Parse individual filings from URLs or files
2. **Batch mode** - Fetch and parse multiple filings by CIK with filtering

### Quick Examples

```bash
# Required: Set your email (SEC requirement), note that "example.com" will be rejected. Email can also be set via CLI flag
export SEC_EMAIL="your-email@example.com"

# Parse single Form 4 from URL
./goedgar https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml

# Parse single Schedule 13D from URL
./goedgar https://www.sec.gov/Archives/edgar/data/1263508/000110465924031033/tm248032d1_sc13d.htm

# Parse 10-K from local file
./goedgar ./moderna_10k.htm

# Fetch all Form 4s for a company (excludes amendments)
./goedgar --cik 1601830 --form 4

# Fetch all Schedule 13D/G filings for a company (includes amendments)
./goedgar --cik 1263508 --form 13

# List filings without downloading (fast preview)
./goedgar --cik 1263508 --form 13D --list-only

# Output to stdout instead of file
./goedgar --cik 1601830 --form 4 -o -

# Date range filtering
./goedgar --cik 1601830 --form 4 --from 2025-01-01 --to 2025-06-30
```

### Single File Mode

Parse individual filings from URLs or local files:

```bash
# Parse from SEC URL
./goedgar https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml

# Parse from local file
./goedgar ./form4.xml

# Save both original filing and JSON output
./goedgar -s https://www.sec.gov/.../ownership.xml

# Specify output file
./goedgar -o results.json https://www.sec.gov/.../ownership.xml

# Output to stdout (pipe to jq, etc.)
./goedgar -o - https://www.sec.gov/.../ownership.xml | jq '.transactions[0]'
```

**Auto-detection:** The parser automatically detects whether the file is Form 4, Schedule 13D/G, or XBRL (10-K/10-Q).

**Output:** Saves to `./output/` by default with smart naming based on CIK and accession number.
Example: `./output/1631574-0001193125-25-314736_ownership.json`

### Batch Mode: Fetch Multiple Filings by CIK

Fetch and parse all filings for a company matching specific criteria:

```bash
# All Form 4s for a CIK
./goedgar --cik 1601830 --form 4
# Saves to: ./output/form4_1601830.json

# All Schedule 13D filings (includes amendments: SC 13D + SC 13D/A)
./goedgar --cik 1263508 --form 13D
# Saves to: ./output/form13D_1263508.json

# All Schedule 13G filings (includes amendments: SC 13G + SC 13G/A)
./goedgar --cik 1263508 --form 13G

# All Schedule 13 filings (13D + 13G + all amendments)
./goedgar --cik 1263508 --form 13
# Saves to: ./output/form13_1263508.json

# Form 4s with date range
./goedgar --cik 1601830 --form 4 --from 2025-01-01 --to 2025-06-30
# Saves to: ./output/2025-01-01_2025-06-30_form4_1601830.json

# Custom output path
./goedgar --cik 1601830 --form 4 -o my_data.json

# Output to stdout (for piping)
./goedgar --cik 1601830 --form 4 -o - | jq '.[] | select(.has10b51Plan == true)'

# Include all historical filings (with pagination - can be slow)
./goedgar --cik 78003 --form 4 --all
```

### List-Only Mode: Preview Without Downloading

Preview what filings are available without downloading and parsing them:

```bash
# List all Form 4s for a CIK (fast)
./goedgar --cik 1263508 --form 4 --list-only
# Saves to: ./output/form4_1263508.json
# Output: [{form, filingDate, accessionNumber, url, ...}, ...]

# List all Schedule 13D/G filings
./goedgar --cik 1263508 --form 13 --list-only

# List with date filtering
./goedgar --cik 1263508 --form 4 --from 2024-01-01 --list-only

# Pipe to jq for analysis
./goedgar --cik 1263508 --form 13 --list-only -o - | jq 'length'
# Output: 433
```

**List-only output:** Returns filing metadata only (form type, filing date, accession number, URL, etc.) without downloading or parsing the actual filings. Useful for:
- Checking what's available before downloading
- Building filing inventories
- Fast filtering and counting

### Form Filtering Behavior

**Important:** Amendment handling differs by form type:

| Form Type | Filter | Matches | Includes Amendments? |
|-----------|--------|---------|---------------------|
| Form 4 | `--form 4` | Form 4 only | ❌ NO (exact match) |
| Form 4 Amendment | `--form 4/A` | Form 4/A only | N/A |
| Schedule 13D | `--form 13D` | SC 13D + SC 13D/A | ✅ YES |
| Schedule 13G | `--form 13G` | SC 13G + SC 13G/A | ✅ YES |
| Schedule 13 (wildcard) | `--form 13` | All Schedule 13 forms | ✅ YES (13D + 13G + amendments) |

### Output Directory

By default, files are saved to `./output/` with smart naming:

**Single file mode:**
- Format: `{CIK}-{ACCESSION}_{filename}.json`
- Example: `1631574-0001193125-25-314736_ownership.json`

**Batch mode:**
- No date range: `form{TYPE}_{CIK}.json`
  - Example: `form4_1601830.json`
- With date range: `{FROM}_{TO}_form{TYPE}_{CIK}.json`
  - Example: `2025-01-01_2025-06-30_form4_1601830.json`

**List-only mode:** Same naming as batch mode.

### Batch Mode Features

- Automatically saves to `./output/` with smart naming
- Automatically fetches company submissions index
- Filters by form type and date range
- Handles pagination for companies with many filings
- Rate-limited to comply with SEC guidelines (10 req/sec)
- Progress indicators during download
- Returns JSON array of all matching filings

**Output format:** Batch mode returns a JSON array where each element has the same structure as single-file mode, making it easy to process both uniformly.

## Form Documentation

### Form 4: Insider Trading

**What is Form 4?**
Form 4 is filed by company insiders (officers, directors, 10%+ owners) within 2 business days of transactions. Tracks insider buying, selling, option exercises, grants, etc.

#### JSON Output Format

Form 4 filings are converted to clean, table-like JSON optimized for data analysis.

**Metadata Section:**

Every Form 4 output includes metadata:

```json
{
  "metadata": {
    "cik": "0001601830",
    "accessionNumber": "0001856369-25-000018",
    "formType": "4",
    "periodOfReport": "2025-12-19",
    "filingDate": "2025-12-19",
    "reportDate": "2025-12-19",
    "source": "https://www.sec.gov/Archives/edgar/data/..."
  },
  "issuer": {...},
  "reportingOwners": [...],
  "transactions": [...]
}
```

**Transaction Structure:**

Each transaction contains these fields:

| Field | Type | Description |
|-------|------|-------------|
| `securityTitle` | string | Security name (e.g., "Common Stock") |
| `transactionDate` | string | Date in YYYY-MM-DD format |
| `transactionCode` | string | Single-letter code (see codes below) |
| `shares` | float64 or null | Number of shares |
| `pricePerShare` | float64 or null | Price per share |
| `acquiredDisposed` | string | "A" (acquired) or "D" (disposed) |
| `sharesOwnedFollowing` | float64 or null | Total shares owned after |
| `directIndirect` | string | "D" (direct) or "I" (indirect) ownership |
| `equitySwapInvolved` | boolean | Equity swap involved |
| `is10b51Plan` | boolean | Under Rule 10b5-1 trading plan |
| `plan10b51AdoptionDate` | string or null | Plan adoption date (YYYY-MM-DD) |
| `footnotes` | array | Footnote IDs (e.g., ["F1"]) |

**Derivative-specific fields:**

| Field | Type | Description |
|-------|------|-------------|
| `exercisePrice` | float64 or null | Strike price for options |
| `exerciseDate` | string | Date exercisable |
| `expirationDate` | string | Expiration date |
| `underlyingTitle` | string | Underlying security |
| `underlyingShares` | float64 or null | Number of underlying shares |

**Transaction Codes:**

| Code | Description | Example |
|------|-------------|---------|
| **P** | Open Market Purchase | Insider bought on market |
| **S** | Open Market Sale | Insider sold on market |
| **A** | Grant/Award/Acquisition | Stock grants, option grants |
| **M** | Exercise of Derivative | Option exercised |
| **F** | Tax Withholding | Shares withheld for taxes |
| **X** | In-the-Money Exercise | Cashless exercise |

See full documentation in [previous README sections] for examples and complete code list.

### Schedule 13D/G: 5%+ Ownership Filings

When an investor acquires 5%+ of a company's stock, they must file:
- **Schedule 13D** - Active/Activist investor (plans to influence control, board seats, strategy)
- **Schedule 13G** - Passive investor (just investing, no control intent)

**13G → 13D transition = activist campaign**

**Key difference:** Item 4 in Schedule 13D describes activist intent ("Purpose of Transaction"). This is where activists outline their demands, criticisms, and plans.

#### JSON Output Format

```json
{
  "formType": "SC 13D",
  "issuerName": "vTv Therapeutics Inc.",
  "issuerCIK": "0001263508",
  "issuerCUSIP": "918385204",
  "securityTitle": "Class A Common Stock, par value $0.01 per share",
  "isAmendment": true,
  "amendmentNumber": 3,
  "filingDate": "2024-03-15",
  "reportingPersons": [
    {
      "name": "Baker Bros. Advisors LP",
      "cik": "0001365204",
      "sharesOwned": 8234567,
      "percentOfClass": 12.5,
      "votingPower": 12.5,
      "dispositivePower": 12.5,
      "memberOfGroup": ""
    }
  ],
  "Items13D": {
    "Item1SecurityTitle": "Class A Common Stock",
    "Item2FilingPersons": "Baker Bros. Advisors LP...",
    "Item3SourceOfFunds": "Working capital",
    "Item4PurposeOfTransaction": "The Reporting Persons purchased the Common Stock for investment purposes and believe the Issuer is significantly undervalued... [7,815 characters of activist intent]",
    "Item5PercentageOfClass": "12.5%",
    "Item6Contracts": "None",
    "Item7Exhibits": "Exhibit 1..."
  }
}
```

**Key fields:**
- `formType` - "SC 13D", "SC 13D/A", "SC 13G", or "SC 13G/A"
- `isAmendment` - Whether this is an amendment
- `amendmentNumber` - Amendment number (1, 2, 3, etc.)
- `reportingPersons` - Array of investors (can be multiple for joint filings)
  - `sharesOwned` - Total shares owned
  - `percentOfClass` - Ownership percentage
  - `votingPower` - Voting power percentage
  - `dispositivePower` - Power to dispose of shares
  - `memberOfGroup` - Joint filer group designation
- `Items13D` - Schedule 13D items (activist filings)
  - **`Item4PurposeOfTransaction`** Activist intent, demands, criticisms
- `Items13G` - Schedule 13G items (passive filings)

### XBRL: 10-K/10-Q Financial Reports

XBRL (eXtensible Business Reporting Language) is the structured format used by the SEC for financial reports (10-K annual reports, 10-Q quarterly reports). The parser extracts key financial metrics into a standardized snapshot.

#### JSON Output Format

```json
{
  "formType": "XBRL",
  "data": {
    "fiscalYearEnd": "2024-12-31",
    "fiscalPeriod": "FY",
    "formType": "10-K",
    "companyName": "Moderna, Inc.",
    "cik": "0001682852",

    "cash": 1930000000,
    "totalAssets": 14140000000,
    "totalLiabilities": 3240000000,
    "stockholdersEquity": 10900000000,
    "totalDebt": 0,

    "revenue": 25000000,
    "netIncome": 3560000000,
    "rdExpense": 4540000000,
    "gaExpense": 1170000000,

    "operatingCashFlow": 3000000000,
    "dilutedShares": 384000000,
    "dilutedEPS": 9.27
  }
}
```

**Extracted metrics (43 GAAP concepts):**

**Balance Sheet:**
- Assets: Cash, A/R, Inventory, Prepaid, PP&E, Intangibles, Goodwill, Total Assets
- Liabilities: Short/Long-Term Debt, A/P, Accrued Liabilities, Deferred Revenue, Total Liabilities
- Equity: Stockholders Equity, Accumulated Deficit, Common Shares Outstanding

**Income Statement:**
- Revenue, COGS, Gross Profit, R&D, G&A, S&M, Operating Expenses, Operating Income, Interest Expense, Tax Expense, Net Income

**Per Share:**
- Basic/Diluted Shares, Basic/Diluted EPS

**Cash Flow:**
- Operating/Investing/Financing Cash Flows, Capex, D&A, Stock-Based Compensation

**Use cases:**
```bash
# Extract cash position for biotech company
./goedgar moderna_10k.htm -o - | jq '{
  company: .data.companyName,
  cash: .data.cash,
  burn: (.data.rdExpense + .data.gaExpense),
  runway: (.data.cash / ((.data.rdExpense + .data.gaExpense) / 4))
}'
```

## Library API

### Quick Start (Auto-Detection)

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    edgar "github.com/RxDataLab/go-edgar"
)

func main() {
    // Auto-detect form type and parse
    // Works with Form 4 XML, Schedule 13D/G HTML/XML, or XBRL
    data := []byte(`<html>...</html>`) // or XML content

    parsed, err := edgar.ParseAny(bytes.NewReader(data))
    if err != nil {
        panic(err)
    }

    fmt.Printf("Form Type: %s\n", parsed.FormType)

    // Export to JSON
    jsonData, _ := json.MarshalIndent(parsed, "", "  ")
    fmt.Println(string(jsonData))
}
```

### Form 4 Specific

```go
import edgar "github.com/RxDataLab/go-edgar"

// Parse Form 4 XML
xmlData := []byte(`<ownershipDocument>...</ownershipDocument>`)
form4, err := edgar.Parse(xmlData)
if err != nil {
    panic(err)
}

// Convert to clean JSON output
output := form4.ToOutput()

// Helper methods
marketTrades := form4.GetMarketTrades() // Only P and S codes
purchases := form4.GetPurchases()       // Only P code
sales := form4.GetSales()               // Only S code
has10b51 := form4.Is10b51Plan()         // Check for trading plan
```

### Schedule 13D/G Specific

```go
import edgar "github.com/RxDataLab/go-edgar"

// Parse Schedule 13D/G (auto-detects HTML vs XML)
data := []byte(`<html>...</html>`)
sc13, err := edgar.ParseSchedule13Auto(data)
if err != nil {
    panic(err)
}

// Check if activist vs passive
isActivist := sc13.IsActivist()  // true for 13D
isPassive := sc13.IsPassive()    // true for 13G

// Access activist intent (13D only)
if sc13.Items13D != nil {
    intent := sc13.Items13D.Item4PurposeOfTransaction
    fmt.Printf("Activist intent: %s\n", intent)
}

// Access ownership info
for _, person := range sc13.ReportingPersons {
    fmt.Printf("%s owns %.1f%%\n", person.Name, person.PercentOfClass)
}
```

### XBRL Specific

```go
import edgar "github.com/RxDataLab/go-edgar"

// Parse XBRL (auto-detects inline vs standalone)
data, _ := os.ReadFile("10k.htm")
xbrl, err := edgar.ParseXBRLAuto(data)
if err != nil {
    panic(err)
}

// Extract financial snapshot
snapshot, err := xbrl.GetSnapshot()
if err != nil {
    panic(err)
}

fmt.Printf("Cash: $%.2fB\n", snapshot.Cash/1e9)
fmt.Printf("R&D: $%.2fB\n", snapshot.RDExpense/1e9)
fmt.Printf("Burn: $%.2fB\n", (snapshot.RDExpense+snapshot.GAExpense)/1e9)
```

### Fetching from SEC

```go
import edgar "github.com/RxDataLab/go-edgar"

// Fetch any form directly from SEC
url := "https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml"
email := "your-email@example.com"

data, err := edgar.FetchForm(url, email)
if err != nil {
    panic(err)
}

// Auto-parse
parsed, _ := edgar.ParseAny(bytes.NewReader(data))
```

### Batch Fetching by CIK

Fetch and parse all filings for a company:

```go
import edgar "github.com/RxDataLab/go-edgar"

opts := edgar.BatchOptions{
    CIK:              "1601830",
    FormType:         "4",        // or "13D", "13G", "13"
    DateFrom:         "2025-01-01",
    DateTo:           "2025-06-30",
    Email:            "your-email@example.com",
    IncludePaginated: false,      // true = fetch all historical
    ListOnly:         false,      // true = metadata only, no parsing
}

result, err := edgar.FetchAndParseBatch(opts)
if err != nil {
    panic(err)
}

fmt.Printf("Found %d filings\n", result.TotalFound)
fmt.Printf("Successfully parsed %d filings\n", result.Fetched)

// Process filings (type depends on FormType)
for _, filing := range result.Filings {
    switch filing.FormType {
    case "4":
        if form4, ok := filing.Data.(*edgar.Form4Output); ok {
            fmt.Printf("Transactions: %d\n", len(form4.Transactions))
        }
    case "SC 13D", "SC 13G":
        if sc13, ok := filing.Data.(*edgar.Schedule13Filing); ok {
            fmt.Printf("Ownership: %.1f%%\n", sc13.ReportingPersons[0].PercentOfClass)
        }
    }
}

// Check for errors
if len(result.Errors) > 0 {
    for _, err := range result.Errors {
        fmt.Printf("Error: %v\n", err)
    }
}
```

### List-Only Mode (Fast Preview)

```go
opts := edgar.BatchOptions{
    CIK:      "1263508",
    FormType: "13",
    Email:    "your-email@example.com",
    ListOnly: true,  // Only fetch metadata, don't parse
}

result, err := edgar.FetchAndParseBatch(opts)

// result.FilingList contains metadata only
for _, filing := range result.FilingList {
    fmt.Printf("%s: %s (%s)\n", filing.FilingDate, filing.Form, filing.AccessionNumber)
}
```

### Core Functions

```go
// Auto-detection and parsing
func ParseAny(r io.Reader) (*ParsedForm, error)

// Form 4
func Parse(data []byte) (*Form4, error)
func (f *Form4) ToOutput() *Form4Output

// Schedule 13D/G
func ParseSchedule13Auto(data []byte) (*Schedule13Filing, error)
func (s *Schedule13Filing) IsActivist() bool
func (s *Schedule13Filing) IsPassive() bool

// XBRL
func ParseXBRLAuto(data []byte) (*XBRL, error)
func DetectXBRLType(data []byte) string
func (x *XBRL) GetSnapshot() (*FinancialSnapshot, error)

// Fetching
func FetchForm(url string, email string) ([]byte, error)
func FetchSubmissions(cik string, email string) (*Submissions, error)
func FetchAndParseBatch(opts BatchOptions) (*BatchResult, error)

// Filtering
func FilterByForm(filings []Filing, formType string) []Filing
func FilterByDateRange(filings []Filing, from, to string) []Filing
```

## Testing

The project uses data-driven tests with JSON ground truth:

```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestForm4Parser/wave_derivatives

# Update golden files after changes
go test -v -run TestForm4Parser -update

# Review changes before accepting
make snapshot-review
make snapshot-accept  # or snapshot-reject
```

See [TESTING.md](TESTING.md) for detailed testing documentation.

## Performance

Typical performance on modern hardware:
- **Parse time**: ~0.5ms per Form 4, ~2ms per Schedule 13D
- **XBRL**: ~100ms for 2.6MB Moderna 10-K (744 facts)
- **Memory**: ~100KB per Form 4 object
- **Throughput**: 2000+ forms/second

## Development

### Project Structure

```
go-edgar/
├── cmd/goedgar/          # CLI tool (all form types)
├── testdata/
│   ├── form4/            # Form 4 test cases
│   ├── schedule13/       # Schedule 13D/G test cases
│   ├── xbrl/             # XBRL test cases
│   └── cik/              # CIK JSON test data
│
├── Form-specific files:
├── form4.go              # Form 4 parsing
├── form4_output.go       # Form 4 JSON output
├── form4_tenb51.go       # 10b5-1 detection
├── schedule13.go         # Schedule 13D/G data structures
├── schedule13_html.go    # Schedule 13 HTML parser
├── xbrl.go               # XBRL core structs
├── xbrl_ixbrl.go         # Inline XBRL parser
├── xbrl_concepts.go      # Concept mappings
├── xbrl_financials.go    # Financial snapshot
│
├── Common utilities:
├── parser.go             # Auto-detection
├── fetcher.go            # SEC HTTP client
├── metadata.go           # File naming
├── submissions.go        # CIK filtering
├── batch.go              # Batch orchestration
└── normalize.go          # Text normalization
```

### Building

```bash
# Build CLI
make build

# Run tests
make test

# Clean artifacts
make clean

# Install to $GOPATH/bin
make install
```

## Related Projects

- [edgartools](https://github.com/dgunning/edgartools) - Python library for SEC filings

## Resources

- [SEC Form 4 Information](https://www.sec.gov/files/form4data.pdf)
- [Schedule 13D/G Guide](https://www.sec.gov/files/schedule13d.pdf)
- [XBRL Resources](https://www.sec.gov/structureddata/osd-inline-xbrl.html)
