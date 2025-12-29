# go-edgar

Fast, zero-dependency Go parser for SEC filings. Used by [RxDataLab](https://rxdatalab.com/) for company analysis and in our data pipelines for [our biotech research application](https://app.rxdatalab.com/)

## Roadmap

**Parsing**
- [X] Form 4
- [ ] 10-K
- [ ] 10-Q
- [ ] 13F
- [ ] Form D
- [ ] 8-K (parse form type e.g., 7.1 and content + links included for later parsing/interpreting)

**CLI Functions**
- [ ] Retrieve and parse supported forms or target form over date range

## Features

- **Zero dependencies** - Uses only Go stdlib for core functionality
- **Type-safe** - Compile-time checked structs
- **SEC fetcher** - Built-in HTTP client with rate limiting
- **CLI tool** - Standalone binary for parsing and fetching filings
- **Fast** - Parse thousands of filings per second
- **Simple API** - Easy to use and understand
- **JSON export** - Clean, table-like output format

## Currently Supported: Form 4

**Form 4** - Insider trading filings by company insiders (officers, directors, 10%+ owners) filed within 2 business days of transactions.

**Form 4-specific features:**
- Complete parsing of non-derivative AND derivative transactions (options, warrants)
- Automatic 10b5-1 trading plan detection with adoption dates
- Transaction filtering (purchases, sales, market trades)
- Footnote parsing and reference resolution

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

The `goedgar` CLI tool parses SEC filings from URLs or files and outputs clean JSON.

### Basic Usage

```bash
# Parse from SEC URL (requires SEC_EMAIL environment variable)
export SEC_EMAIL="your-email@example.com"
./goedgar https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml

# Parse from local file
./goedgar ./form4.xml

# Save both original filing and JSON output
./goedgar -s https://www.sec.gov/.../ownership.xml

# Specify output file
./goedgar -o results.json https://www.sec.gov/.../ownership.xml
```

### Output Directory

By default, files are saved to `./output/` with smart naming based on CIK and accession number.

Example: `1631574-0001193125-25-314736_ownership.json`

## Form 4 Documentation

### JSON Output Format

Form 4 filings are converted to clean, table-like JSON optimized for data analysis.

### Transaction Structure

Each transaction (non-derivative and derivative) contains these fields:

| Field | Type | Description |
|-------|------|-------------|
| `securityTitle` | string | Security name (e.g., "Common Stock", "Share Option (right to buy)") |
| `transactionDate` | string | Date in YYYY-MM-DD format |
| `transactionCode` | string | Single-letter code (see Transaction Codes table below) |
| `shares` | float64 or null | Number of shares, null if not applicable |
| `pricePerShare` | float64 or null | Price per share, null for exercises or grants |
| `acquiredDisposed` | string | "A" (acquired) or "D" (disposed) |
| `sharesOwnedFollowing` | float64 or null | Total shares owned after transaction |
| `directIndirect` | string | "D" (direct) or "I" (indirect) ownership |
| `equitySwapInvolved` | boolean | Whether transaction involved equity swap |
| `is10b51Plan` | boolean | Whether transaction is under Rule 10b5-1 trading plan |
| `plan10b51AdoptionDate` | string or null | Date plan was adopted (YYYY-MM-DD), null if not applicable |
| `footnotes` | array | Array of footnote IDs (e.g., ["F1", "F2"]) |

**Derivative-specific fields:**
| Field | Type | Description |
|-------|------|-------------|
| `exercisePrice` | float64 or null | Strike price for options/warrants |
| `exerciseDate` | string | Date option becomes exercisable |
| `expirationDate` | string | Date option expires |
| `underlyingTitle` | string | Underlying security (e.g., "Common Shares") |
| `underlyingShares` | float64 or null | Number of underlying shares |

### Transaction Codes

| Code | Description | Typical Use |
|------|-------------|-------------|
| **P** | Open Market Purchase | Insider bought stock on public market |
| **S** | Open Market Sale | Insider sold stock on public market |
| **A** | Grant, Award or Other Acquisition | Stock grants, option grants, or acquisitions |
| **D** | Disposition to the Issuer | Shares returned to company |
| **F** | Payment of Exercise Price or Tax Liability | Shares withheld for taxes |
| **G** | Gift | Shares gifted to another party |
| **M** | Exercise or Conversion of Derivative Security | Option or warrant exercised |
| **C** | Conversion of Derivative Security | Convertible security converted |
| **X** | Exercise of In-the-Money or At-the-Money Derivative Security | Cashless exercise of derivative |

### Acquired/Disposed Codes

| Code | Meaning |
|------|---------|
| **A** | Acquired - Insider gained shares |
| **D** | Disposed - Insider lost/sold shares |

## Understanding Output

### Form 4

#### Simple 10b5-1 Sale (Wave Life Sciences)

A straightforward insider sale under a pre-arranged trading plan:

```json
{
  "formType": "4",
  "periodOfReport": "2025-12-08",
  "has10b51Plan": true,
  "issuer": {
    "cik": "0001631574",
    "name": "Wave Life Sciences Ltd.",
    "ticker": "WVE"
  },
  "transactions": [
    {
      "securityTitle": "Ordinary Shares",
      "transactionDate": "2025-12-08",
      "transactionCode": "S",
      "shares": 60000,
      "pricePerShare": 13.2,
      "acquiredDisposed": "D",
      "sharesOwnedFollowing": 89218,
      "directIndirect": "D",
      "equitySwapInvolved": false,
      "is10b51Plan": true,
      "plan10b51AdoptionDate": "2025-03-13",
      "footnotes": ["F1"]
    }
  ]
}
```

**What this means:**
- CFO sold 60,000 shares at $13.20 per share
- Sale was pre-planned under Rule 10b5-1 (adopted March 13, 2025)
- After sale, CFO owns 89,218 shares
- This was a direct ownership transaction (not held in trust)

#### Complex Warrant Exercise (ProMis Neurosciences)

A more complex scenario showing warrant exercises. This demonstrates why some fields are `null`:

```json
{
  "formType": "4",
  "periodOfReport": "2025-07-25",
  "has10b51Plan": false,
  "issuer": {
    "cik": "0001374339",
    "name": "ProMIS Neurosciences Inc.",
    "ticker": "PMN"
  },
  "transactions": [
    {
      "securityTitle": "Common Shares, no par value",
      "transactionDate": "2025-07-25",
      "transactionCode": "X",
      "shares": 697674,
      "pricePerShare": null,
      "acquiredDisposed": "A",
      "sharesOwnedFollowing": 2315111,
      "directIndirect": "D",
      "equitySwapInvolved": false,
      "is10b51Plan": false,
      "plan10b51AdoptionDate": null,
      "footnotes": ["F1"]
    }
  ],
  "derivatives": [
    {
      "securityTitle": "Tranche A Common Share Purchase Warrants",
      "transactionDate": "2025-07-25",
      "transactionCode": "X",
      "shares": 697674,
      "pricePerShare": 0,
      "acquiredDisposed": "D",
      "exercisePrice": 2.02,
      "underlyingTitle": "Common Shares",
      "underlyingShares": 697674,
      "sharesOwnedFollowing": 0,
      "directIndirect": "D",
      "equitySwapInvolved": false,
      "is10b51Plan": false,
      "plan10b51AdoptionDate": null,
      "footnotes": ["F1"]
    }
  ],
  "footnotes": [
    {
      "id": "F1",
      "text": "On July 25, 2025, the Jeremy M. Sclar 2012 Irrevocable Family Trust exercised 697,674 Tranche A purchase warrants, each exercisable to purchase one Common Share. These warrants were exercisable at an exercise price of $2.02 per warrant share; however, following an offer by the JS Trust and an acceptance by the Issuer, were exercised in full at an exercise price of $0.83518 per share."
    }
  ]
}
```

**What this means:**
- **Transaction (code X)**: Holder ACQUIRED 697,674 common shares by exercising warrants
  - `pricePerShare: null` - Not an open market purchase, so no market price
  - `acquiredDisposed: "A"` - Acquired shares
  - After exercise, holder owns 2,315,111 shares total

- **Derivative (code X)**: Holder DISPOSED of 697,674 warrants by exercising them
  - `pricePerShare: 0` - Exercise has nominal transaction price
  - `acquiredDisposed: "D"` - Disposed of warrants (they were consumed in the exercise)
  - `exercisePrice: 2.02` - Original strike price (see footnote for actual price paid)
  - `sharesOwnedFollowing: 0` - No warrants left after exercise

**Why code "X" instead of "M"?**
- **M** = Standard option exercise
- **X** = In-the-money or at-the-money exercise (often cashless or reduced-price)

**Why `pricePerShare: null` in transactions but `0` in derivatives?**
- Transactions: `null` means no market price (shares acquired via exercise, not purchase)
- Derivatives: `0` is nominal price for the derivative transaction itself

## Library API

### Quick Start

Note that only Form 4 is supported at this time.

```go
package main

import (
    "fmt"
    "encoding/json"
    edgar "github.com/RxDataLab/go-edgar"
)

func main() {
    // Parse from XML string or file
    xmlData := []byte(`<ownershipDocument>...</ownershipDocument>`)
    form4, err := edgar.Parse(xmlData)
    if err != nil {
        panic(err)
    }

    // Convert to clean JSON output format
    output := form4.ToOutput()

    // Export to JSON
    jsonData, _ := json.MarshalIndent(output, "", "  ")
    fmt.Println(string(jsonData))
}
```

### Fetching from SEC

```go
import edgar "github.com/RxDataLab/go-edgar"

// Fetch Form 4 directly from SEC
// Note: SEC requires User-Agent with email
edgar.SetUserEmail("your-email@example.com")

url := "https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml"
xmlData, err := edgar.FetchForm(url)
if err != nil {
    panic(err)
}

// Parse fetched data
form4, _ := edgar.Parse(xmlData)
output := form4.ToOutput()
```

### Core Functions

```go
// Parse Form 4 XML into raw struct
func Parse(data []byte) (*Form4, error)

// Convert Form4 to clean JSON output format
func (f *Form4) ToOutput() *Form4Output

// Fetch Form 4 from SEC (with rate limiting)
func FetchForm(url string) ([]byte, error)

// Set email for SEC User-Agent (required by SEC)
func SetUserEmail(email string)
```

### Helper Methods (on raw Form4 struct)

```go
// Get only open market purchases and sales
func (f *Form4) GetMarketTrades() []NonDerivativeTransaction

// Get only purchases
func (f *Form4) GetPurchases() []NonDerivativeTransaction

// Get only sales
func (f *Form4) GetSales() []NonDerivativeTransaction

// Check if form has Rule 10b5-1 trading plan
func (f *Form4) Is10b51Plan() bool
```

### Transaction Code Descriptions

```go
// Get human-readable description of transaction code
description := edgar.TransactionCodeDescription("S")
// Returns: "Open Market Sale"
```

## Working with Raw Data

The library preserves the original XML structure for advanced use cases:

```go
// Access raw Form4 struct
form4, _ := edgar.Parse(xmlData)

// Access issuer info
fmt.Printf("Company: %s (%s)\n", form4.Issuer.Name, form4.Issuer.TradingSymbol)

// Access reporting owner
owner := form4.ReportingOwners[0]
fmt.Printf("Insider: %s\n", owner.ID.Name)
fmt.Printf("Title: %s\n", owner.Relationship.OfficerTitle)

// Access transactions
for _, txn := range form4.NonDerivativeTable.Transactions {
    // Values are stored as strings in raw struct
    fmt.Printf("Shares: %s\n", txn.Amounts.Shares.Value)

    // Convert to numeric types when needed
    shares, _ := txn.Amounts.Shares.Int()
    price, _ := txn.Amounts.PricePerShare.Float64()
}

// Access footnotes
for _, fn := range form4.Footnotes {
    fmt.Printf("%s: %s\n", fn.ID, fn.Text)
}
```

## Testing

The project uses data-driven tests with JSON ground truth files and automatic schema validation.

```bash
# Run all tests
go test -v

# Run specific test case
go test -v -run TestForm4Parser/wave_derivatives

# Update golden files after parser changes
go test -v -run TestForm4Parser -update

# Review snapshot changes before accepting
make snapshot-review
make snapshot-accept  # or snapshot-reject

# Run benchmarks
go test -bench=.
```

See [TESTING.md](TESTING.md) and [SNAPSHOT_TESTING.md](SNAPSHOT_TESTING.md) for detailed testing documentation.

## Performance

Typical performance on modern hardware:
- **Parse time**: ~0.5ms per Form 4
- **Memory**: ~100KB per Form 4 object
- **Throughput**: 2000+ forms/second

## Development

### Project Structure

```
go-edgar/
├── cmd/goedgar/          # CLI tool
├── testdata/form4/       # Test cases with ground truth
├── form4.go              # Core parsing logic
├── form4_output.go       # JSON output format
├── tenb51.go             # 10b5-1 detection logic
├── fetcher.go            # SEC HTTP client
├── parser.go             # Auto-detection and dispatch
└── metadata.go           # File naming and metadata
```

### Building

```bash
# Build CLI
make build

# Run tests
make test

# Build snapshot for release
make snapshot
```

## Related Projects

- [edgartools](https://github.com/dgunning/edgartools) - Python library for SEC filings

## Resources

- [SEC Form 4 Information](https://www.sec.gov/files/form4data.pdf)
