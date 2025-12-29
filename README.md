# go-edgar

Fast, zero-dependency Go parser for SEC Form 4 insider trading filings. Used by [RxDataLab](https://rxdatalab.com/) for company analysis and in our data pipelines for [our biotech research application](https://app.rxdatalab.com/)

## What is Form 4?

SEC Form 4 is filed by company insiders (officers, directors, 10%+ owners) within 2 business days of buying or selling company stock. It's a key dataset for tracking insider trading activity.

## Features

- **Zero dependencies** - Uses only Go stdlib
- **Complete parsing** - Non-derivative AND derivative transactions (options, warrants)
- **10b5-1 detection** - Automatic detection and tagging of Rule 10b5-1 trading plans
- **Type-safe** - Compile-time checked structs
- **SEC fetcher** - Built-in HTTP client with rate limiting
- **Fast** - Parse thousands of filings per second
- **Simple API** - Easy to use and understand

## Installation

```bash
go get github.com/RxDataLab/go-edgar
```

## Quick Start

```go
package main

import (
    "fmt"
    "os"
    "github.com/RxDataLab/go-edgar"
)

func main() {
    // Read Form 4 XML
    xmlData, _ := os.ReadFile("form4.xml")

    // Parse it
    f4, err := form4.Parse(xmlData)
    if err != nil {
        panic(err)
    }

    // Access the data
    fmt.Printf("Company: %s (%s)\n", f4.Issuer.Name, f4.Issuer.TradingSymbol)
    fmt.Printf("Insider: %s\n", f4.ReportingOwners[0].ID.Name)
    fmt.Printf("Title: %s\n", f4.ReportingOwners[0].Relationship.OfficerTitle)

    // Get market trades (purchases and sales)
    for _, trade := range f4.GetMarketTrades() {
        fmt.Printf("%s: %s shares at $%s (%s)\n",
            trade.TransactionDate,
            trade.Amounts.Shares.Value,
            trade.Amounts.PricePerShare.Value,
            form4.TransactionCodeDescription(trade.Coding.Code),
        )
    }
}
```

## Transaction Codes

Form 4 uses single-letter codes to indicate transaction types:

| Code | Description |
|------|-------------|
| **P** | Open Market Purchase |
| **S** | Open Market Sale |
| **A** | Grant/Award |
| **M** | Exercise of Option |
| **F** | Tax Payment |
| **G** | Gift |
| **D** | Disposition to Issuer |

See `TransactionCodeDescription()` for the complete mapping.

## API

### Core Functions

```go
// Parse Form 4 XML
func Parse(data []byte) (*Form4, error)

// Get transaction code description
func TransactionCodeDescription(code string) string
```

### Form4 Methods

```go
// Get only open market purchases and sales
func (f *Form4) GetMarketTrades() []NonDerivativeTransaction

// Get only purchases
func (f *Form4) GetPurchases() []NonDerivativeTransaction

// Get only sales
func (f *Form4) GetSales() []NonDerivativeTransaction

// Check if form is under Rule 10b5-1 trading plan
func (f *Form4) Is10b51Plan() bool

// Get 10b5-1 plan adoption date from footnotes
func (f *Form4) Get10b51AdoptionDate() string

// Check if specific transaction is under 10b5-1 plan
func (t *NonDerivativeTransaction) IsUnder10b51(form *Form4) bool
```

### Value Conversion Methods

```go
// Convert string values to numeric types
func (v Value) Float64() (float64, error)  // For prices, shares
func (v Value) Int() (int, error)          // For whole numbers
```

### SEC Fetcher

```go
// Fetch Form 4 XML directly from SEC
func FetchForm(url string) ([]byte, error)
```

### Data Export

```go
import "encoding/json"

// Export to JSON
jsonData, _ := json.MarshalIndent(f4, "", "  ")
fmt.Println(string(jsonData))
```

## Advanced Usage

### Derivative Transactions

```go
// Access derivative transactions (options, warrants, etc.)
if f4.DerivativeTable != nil {
    for _, deriv := range f4.DerivativeTable.Transactions {
        fmt.Printf("Option: %s\n", deriv.SecurityTitle)

        // Convert exercise price to float
        price, _ := deriv.ConversionOrExercisePrice.Float64()
        fmt.Printf("Exercise Price: $%.2f\n", price)

        // Access underlying security
        shares, _ := deriv.UnderlyingSecurity.Shares.Int()
        fmt.Printf("Underlying: %s (%d shares)\n",
            deriv.UnderlyingSecurity.SecurityTitle.Value, shares)
    }
}
```

### 10b5-1 Trading Plan Detection

```go
// Check if filing is under Rule 10b5-1 trading plan
if f4.Is10b51Plan() {
    fmt.Println("Filed under 10b5-1 trading plan")

    // Get plan adoption date
    if date := f4.Get10b51AdoptionDate(); date != "" {
        fmt.Printf("Plan adopted on: %s\n", date)
    }
}

// Check individual transactions
for _, txn := range f4.NonDerivativeTable.Transactions {
    if txn.IsUnder10b51(f4) {
        fmt.Printf("Transaction %s is under 10b5-1 plan\n", txn.Coding.Code)
    }
}
```

### Fetching from SEC

```go
import "github.com/RxDataLab/go-edgar"

// Fetch Form 4 directly from SEC
url := "https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml"
xmlData, err := edgar.FetchForm(url)
if err != nil {
    panic(err)
}

// Parse fetched data
f4, _ := edgar.Parse(xmlData)
fmt.Printf("Fetched: %s\n", f4.Issuer.Name)
```

### Numeric Value Conversions

```go
// Values are stored as strings to preserve precision
// Convert to numbers when needed
for _, txn := range f4.GetMarketTrades() {
    // Convert shares to int
    shares, err := txn.Amounts.Shares.Int()
    if err != nil {
        // Handle footnote-only or empty values
        fmt.Printf("Shares: %s (footnote: %s)\n",
            txn.Amounts.Shares.Value,
            txn.Amounts.Shares.Footnote())
        continue
    }

    // Convert price to float64
    price, _ := txn.Amounts.PricePerShare.Float64()

    total := float64(shares) * price
    fmt.Printf("Total value: $%.2f\n", total)
}
```

## Data Structure

```go
type Form4 struct {
    DocumentType       string
    PeriodOfReport     string
    Issuer             Issuer
    ReportingOwners    []ReportingOwner
    NonDerivativeTable *NonDerivativeTable  // Common stock
    DerivativeTable    *DerivativeTable     // Options, warrants
    Footnotes          []Footnote
    Signatures         []Signature
}

type NonDerivativeTransaction struct {
    SecurityTitle   string              // e.g., "Class A Common Stock"
    TransactionDate string              // YYYY-MM-DD
    Coding          TransactionCoding   // Code, form type
    Amounts         TransactionAmounts  // Shares, price
    // ... more fields
}
```

## Testing

The project uses data-driven tests with JSON ground truth files. See [TESTING.md](TESTING.md) for details.

```bash
# Run all tests
go test -v

# Run specific test case
go test -v -run TestForm4Parser/snow

# Run benchmarks
go test -bench=.

# Run with coverage
go test -cover
```

### Adding Test Cases

New test cases are easy to add. Just create a directory in `testdata/form4/` with:
- `input.xml` - The Form 4 XML file
- `expected.json` - Expected parsed output with metadata (source_url, notes)

Tests automatically discover and run all cases. See `testdata/form4/README.md` for details.

## Roadmap

- [x] Form 4 XML parsing
- [x] Basic transaction filtering
- [ ] Derivative transaction support
- [ ] Footnote resolution
- [ ] SEC API client (fetch filings)
- [ ] CSV export
- [ ] CLI tool

## Performance

Typical performance on modern hardware:
- **Parse time**: ~1ms per Form 4
- **Memory**: ~100KB per Form 4 object
- **Throughput**: 1000+ forms/second

## Related Projects

- [edgartools](https://github.com/dgunning/edgartools) - Python library for SEC filings (reference implementation)

## Resources

- [SEC Form 4 Information](https://www.sec.gov/files/form4data.pdf)
- [SEC EDGAR Search](https://www.sec.gov/edgar/searchedgar/companysearch.html)
