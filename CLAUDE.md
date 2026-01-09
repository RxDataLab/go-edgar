# go-edgar Development Guide

## Goal

Extract SEC form data (Form 4, Schedule 13D/G, and eventually other forms) from XML filings into Go structs.

**Dual-purpose design:**
- **Library**: Import and use programmatically in other Go tools
- **CLI**: Standalone tool for parsing and fetching SEC filings

Simple, fast, minimal dependencies (stdlib only for core functionality).

## Parsing

- Parse Form 4 XML using stdlib `encoding/xml`
- Map to clean Go structs
- Provide helper methods (GetMarketTrades, GetPurchases, GetSales)
- Export to JSON
- Keep it simple and fast

## Project Structure

```
go-edgar/
├── cmd/
│   └── goedgar/
│       └── main.go         # CLI entry point (handles all form types)
├── form4.go                # Form 4 parsing logic & data structures
├── form4_output.go         # Form 4 JSON output format
├── form4_tenb51.go         # Form 4 10b5-1 detection logic
├── form4_test.go           # Data-driven tests
├── schedule13.go           # Schedule 13D/G parsing logic & data structures
├── schedule13_test.go      # Schedule 13D/G tests
├── xbrl.go                 # XBRL core structs (Fact, Context, Period, Unit)
├── xbrl_ixbrl.go           # Inline XBRL (iXBRL) parser
├── xbrl_concepts.go        # Concept mapping system (XBRL → standardized labels)
├── xbrl_financials.go      # Query interface & high-level helpers
├── xbrl_test.go            # XBRL tests (Moderna 10-K)
├── concept_mappings.json   # Embedded concept mappings (biotech-focused)
├── parser.go               # Auto-detect form type & dispatch
├── fetcher.go              # SEC HTTP client with rate limiting
├── metadata.go             # Metadata extraction & file naming
├── submissions.go          # CIK JSON parsing and filtering
├── batch.go                # Batch download orchestration
├── testdata/
│   ├── form4/              # Form 4 test cases (auto-discovered)
│   │   ├── README.md
│   │   ├── snow/
│   │   ├── arrowhead_footnotes/
│   │   └── wave_derivatives/
│   └── xbrl/               # XBRL test cases
│       └── moderna_10k/    # Moderna FY2024 10-K (inline XBRL)
├── output/                 # Default output directory for CLI
├── scripts/                # Validation & testing scripts
│   ├── generate_edgartools_reference.py
│   ├── fetch_tiingo.py
│   └── fetch_alphavantage.sh
├── go.mod                  # Go module definition
├── Makefile                # Build automation
├── CLAUDE.md               # This file (AI context)
├── README.md               # User-facing docs
├── TESTING.md              # Detailed testing documentation
├── TESTING_VALIDATION.md   # Cross-validation methodology
└── XBRL.md                 # XBRL parsing documentation
```

## Current Status

**Phase 1: Foundation** ✅ COMPLETE
- [x] Core structs defined
- [x] Basic XML parsing implemented
- [x] Helper methods added
- [x] Tests passing with real data (Snowflake Form 4)
- [x] Go module initialized

**Phase 2: Features** ✅ COMPLETE
- [x] Derivative transaction support (options, warrants, exercise prices)
- [x] Numeric value parsing (Float64/Int methods with error handling)
- [x] 10b5-1 trading plan detection (XML flag + footnote regex)
- [x] SEC fetcher with rate limiting and proper User-Agent

**Phase 3: CLI & Multi-Form Support** ✅ COMPLETE
- [x] CLI tool (`goedgar`) with URL and file path support
- [x] Auto-detect form type (currently supports Form 4)
- [x] Smart file naming (CIK-accession_ownership.xml/json)
- [x] SEC email validation (required via --email flag or SEC_EMAIL env var)
- [x] Save original XML with --save-original flag
- [x] Output to ./output/ directory by default
- [ ] Support Form 3 and Form 5 (same XML schema as Form 4)
- [ ] Support Form 13F
- [ ] Query by CIK with date range filtering
- [ ] Batch download mode

**Phase 4: 10-K/10-Q XBRL Parsing** ✅ COMPLETE
- [x] Inline XBRL (iXBRL) parser for HTML-embedded facts
- [x] Auto-detection (inline vs standalone XBRL)
- [x] Concept mapping system with go:embed (JSON → binary)
- [x] Biotech-focused concept mappings (Cash, R&D, G&A, Burn, etc.)
- [x] Fluent query interface (ByLabel, ByPeriod, InstantOnly, etc.)
- [x] High-level financial helpers (GetCash, GetBurn, GetSnapshot, etc.)
- [x] Tested against Moderna 10-K (FY2024, 2.6 MB, 744 facts)
- [x] Table-like output (FinancialSnapshot)
- [x] Comprehensive documentation (XBRL.md)
- [x] Easy to extend (add mappings to JSON, no code changes)

**Phase 5: Schedule 13D/G Support** ✅ COMPLETE
- [x] Core data structures for 13D and 13G filings
- [x] Schedule 13D parser with all 7 items (including Item 4 - activist intent)
- [x] Schedule 13G parser with all 10 items (including Item 10 - passive certification)
- [x] Amendment detection and tracking (/A suffix, amendment numbers)
- [x] Joint filer aggregation logic (memberOfGroup field, no double-counting)
- [x] Auto-detection via XML namespace
- [x] CLI integration with JSON output
- [x] Comprehensive test suite (edgartools reference data)
- [x] Handle missing CIKs (foreign entities, fallback to filer CIK)
- [x] Edge cases: element name differences (13D vs 13G)
- [ ] HTML parser for legacy filings (pre-2018)

**Phase 6: Polish** (future)
- [ ] CSV export
- [ ] Validation
- [ ] Performance optimization
- [ ] Form 13F support
- [ ] Dimensional data (revenue by segment)
- [ ] Time series queries

## Key Design Decisions

1. **Minimal production dependencies** - Dependencies are fine for testing (testify, go-cmp), but the final prod binary should minimize deps where practical. Use stdlib for all core functionality.
2. **Structs over maps** - Type-safe, compile-time checked
3. **JSON export** - Instead of DataFrames, use JSON/CSV
4. **Direct XML mapping** - Let Go's xml package do the heavy lifting
5. **Preserve raw data** - Keep values as strings, add conversion methods (Float64/Int) that return errors
6. **Library-first design** - All logic lives in the library (metadata extraction, file naming, etc.). CLI is just a thin orchestrator.
7. **Auto-detection** - Single CLI tool that auto-detects form type instead of separate tools per form

## Testing Strategy

**Data-driven testing with JSON ground truth:**

1. Each test case = directory in `testdata/form4/`
2. Contains `input.xml` (real SEC Form 4) + `expected.json` (ground truth)
3. Tests auto-discover and compare parsed output against expected
4. Easy to add new cases - just drop files in a folder
5. Metadata tracks source URL and what each test validates

**Current test cases:**
- `snow/` - Basic Form 4 (Snowflake CFO) with non-derivative and derivative transactions
- `arrowhead_footnotes/` - Edge case: Multiple footnotes, 10b5-1 plan, weighted average prices
- `wave_derivatives/` - Derivative transactions (options), 10b5-1 plan with adoption date parsing

See `TESTING.md` for full documentation.

## Reference

- Python implementation: `/home/nick/projects/port-edgartools/edgartools/edgar/ownership/ownershipforms.py`
- Test data source: `/home/nick/projects/port-edgartools/edgartools/data/form4.snow.xml`
- SEC XML Schema: X0306

## CLI Usage

```bash
# Build
make build

# Parse from URL (requires SEC_EMAIL env var or --email flag)
export SEC_EMAIL="your-email@example.com"
./goedgar https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml

# Parse from file
./goedgar ./ownership.xml

# Save original XML and JSON output (to ./output/ directory)
./goedgar -s https://www.sec.gov/Archives/edgar/data/.../ownership.xml

# Save to specific file
./goedgar -o results.json https://www.sec.gov/.../ownership.xml

# Help
./goedgar --help
```

**Smart file naming:**
- URL: `https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml`
- Saves as: `./output/1631574-0001193125-25-314736_ownership.xml` and `.json`

## Recent Updates

**2026-01-05: Schedule 13D/G Implementation**
- ✅ Complete parsing for SEC Schedule 13D and 13G filings (XML format)
- ✅ Comprehensive data extraction:
  - Issuer information (CIK, name, CUSIP)
  - Reporting persons (CIK, name, ownership, voting/dispositive powers)
  - All narrative items (13D: Items 1-7, 13G: Items 1-10)
- ✅ Amendment tracking with number extraction
- ✅ Joint filer aggregation (memberOfGroup="a" handling)
- ✅ Auto-detection via XML namespace (http://www.sec.gov/edgar/schedule13D vs schedule13g)
- ✅ CLI integration with JSON output
- ✅ Edge case handling:
  - Missing CIKs (fallback to filer CIK)
  - Foreign entities (noCIK flag)
  - Element name differences (percentOfClass vs classPercent)
  - Amendment number variations
- ✅ Comprehensive test suite (all tests passing)
- ⏳ HTML parser for legacy filings (Julian Baker's 2014-2017 filings use HTML)

**Key Features:**
- **Track activist campaigns**: Item 4 (Purpose of Transaction) captures activist intent
- **Detect strategy shifts**: Identify 13G → 13D transitions (passive → activist)
- **Accurate ownership**: Joint filer logic prevents double-counting
- **Amendment history**: Track ownership changes over time

**Files Added:**
- `schedule13.go` (637 lines) - Core parsing and data structures
- `schedule13_test.go` (307 lines) - Comprehensive tests
- Updated `parser.go` - Auto-detection for 13D/G forms
- `parser_test.go` - Form detection tests

**2025-12-31: 10-K/10-Q XBRL Parsing Implementation**
- ✅ Implemented inline XBRL (iXBRL) parser for modern SEC filings
- ✅ Auto-detection between inline and standalone XBRL formats
- ✅ Concept mapping system with `go:embed` (JSON compiled into binary)
- ✅ 14 biotech-focused concept mappings (Cash, R&D, G&A, Burn, Debt, Shares, Revenue, etc.)
- ✅ Fluent query interface: `xbrl.Query().ByLabel("Cash").InstantOnly().MostRecent()`
- ✅ High-level helpers: `GetCash()`, `GetBurn()`, `GetSnapshot()`
- ✅ Table-like output: `FinancialSnapshot` struct with runway calculations
- ✅ Tested against Moderna 10-K FY2024 (2.6 MB, 368 contexts, 744 facts)
- ✅ Extracted metrics: Cash ($1.93B), Revenue ($3.24B), R&D ($4.54B), Burn ($5.72B)
- ✅ Comprehensive documentation in `XBRL.md` (edge cases, troubleshooting, examples)
- ✅ Easy extensibility: add new concepts to `concept_mappings.json`, no code changes needed
- ✅ Zero external dependencies (stdlib only)

**Key Features:**
- **Minimal dataset extraction**: Focus on biotech essentials (Cash, R&D, G&A, Burn, Runway)
- **Structured output**: Table-like `FinancialSnapshot` for easy testing and display
- **Edge case handling**: Decimal scaling, multiple contexts, company extensions, character encoding
- **18.8% mapping rate**: Expected for biotech filings (unmapped = metadata + extensions)

**Files Added:**
- `xbrl.go` - Core data structures (144 lines)
- `xbrl_ixbrl.go` - Inline XBRL parser (177 lines)
- `xbrl_concepts.go` - Concept mapping (99 lines)
- `xbrl_financials.go` - Query interface + helpers (351 lines)
- `xbrl_test.go` - Comprehensive tests (344 lines)
- `concept_mappings.json` - Embedded mappings (87 lines)
- `XBRL.md` - Full documentation (536 lines)
- `testdata/xbrl/moderna_10k/` - Real test data (2.6 MB)

**2025-12-28: Simplified JSON Output Structure**
- ✅ Created `form4_output.go` with table-like JSON structure
- ✅ Numeric types for shares and prices (float64 instead of strings)
- ✅ Per-transaction 10b5-1 detection (`is10b51Plan` on each transaction)
- ✅ Flat structure with clear field names (camelCase)
- ✅ Footnote IDs as arrays (e.g., `["F1", "F4"]`) instead of nested objects
- ✅ Updated tests to validate new output format
- ✅ Regenerated all golden files with simplified structure

**2025-12-28: CLI Implementation**
- ✅ Created unified CLI tool `goedgar` with auto-detection
- ✅ Added `parser.go` for form type auto-detection
- ✅ Added `metadata.go` for CIK/accession extraction and smart file naming
- ✅ Updated `fetcher.go` to require email (SEC_EMAIL env var or --email flag)
- ✅ CLI saves to ./output/ directory by default
- ✅ Library-first design: all logic in library, CLI is thin orchestrator
- ✅ Created Makefile for build automation

**2025-12-28: Test Refactoring & Feature Completion**
- ✅ Migrated to data-driven tests with JSON ground truth
- ✅ Organized test cases in `testdata/form4/` directories
- ✅ Added metadata (source_url, notes) to each test case
- ✅ Tests auto-discover new test cases (no code changes needed)
- ✅ Implemented golden file testing with `-update` flag
- ✅ Completed derivative transaction parsing (options, warrants, underlying securities)
- ✅ Added numeric conversion methods (Float64/Int with error handling)
- ✅ Implemented 10b5-1 trading plan detection (dual approach: XML flag + footnote regex)
- ✅ Created SEC fetcher with rate limiting and proper User-Agent headers
- ✅ Added wave_derivatives test case with comprehensive derivative examples

**2025-12-31: Comprehensive GAAP Field Support**
- ✅ Expanded `concept_mappings.json` from 14 to 43 comprehensive GAAP concepts
- ✅ Added `requiredFields` array to track GAAP-mandated fields
- ✅ Expanded `FinancialSnapshot` struct to include all 43 mapped fields:
  - **Balance Sheet Assets**: Cash, A/R, Inventory, Prepaid, PP&E, Intangibles, Goodwill, Total Assets
  - **Balance Sheet Liabilities**: Short/Long-Term Debt, A/P, Accrued Liabilities, Deferred Revenue, Total Liabilities
  - **Balance Sheet Equity**: Stockholders Equity, Accumulated Deficit, Common Shares Outstanding
  - **Income Statement**: Revenue, COGS, Gross Profit, R&D, G&A, S&M, Operating Expenses, Operating Income, Interest Expense, Tax Expense, Net Income
  - **Per Share Metrics**: Basic/Diluted Shares, Basic/Diluted EPS
  - **Cash Flow Statement**: Operating/Investing/Financing Cash Flows, Capex
  - **Non-Cash Items**: D&A, Stock-Based Compensation
- ✅ Implemented `validateRequiredFields()` to detect missing GAAP-mandated fields
- ✅ Added `MissingRequiredFields` array to snapshot output for debugging tag mappings
- ✅ Refactored `GetSnapshot()` with helper functions for cleaner code
- ✅ Updated tests to use new comprehensive structure
- ✅ CLI now outputs all 43 fields in structured JSON format

**Key Benefits:**
- **Complete financial picture**: No more manual queries for individual metrics
- **Automatic validation**: Identifies missing required fields that indicate incorrect tag mappings
- **Easy extensibility**: Add new concepts to JSON, GetSnapshot() auto-populates
- **Backward compatible**: All existing code continues to work

**2025-12-31: Value Validation Framework** 📋 PLANNED
- 📋 Documented cross-validation methodology against 3 independent sources:
  - **edgartools** (Python reference implementation)
  - **Alpha Vantage** (commercial XBRL parser)
  - **Tiingo** (commercial fundamental data provider)
- 📋 Created test company matrix (10 biotechs with diverse reporting styles)
- 📋 Designed comparison pipeline with ±1% tolerance
- 📋 Identified key edge cases:
  - Foreign filers (ADRs)
  - Pre-revenue companies ($0 revenue)
  - Recent IPOs (incomplete data)
  - Acquisitions/restructuring (goodwill, intangibles)
  - Stock splits (share count adjustments)
  - Quarterly vs annual (YTD context selection)
- 📋 Automated validation workflow (GitHub Actions ready)
- 📋 See `TESTING_VALIDATION.md` for complete methodology

**Implementation Status:**
- ✅ Documentation complete
- ⏳ Scripts to be created (generate_edgartools_reference.py, fetch_tiingo.py, compare.py)
- ⏳ Test data directory structure (testdata/validation/)
- ⏳ GitHub Actions workflow (.github/workflows/validate-xbrl.yml)

## Next Steps

**CLI Enhancements:**
1. Add Form 3/5 support (same schema as Form 4, just different documentType)
2. Add Form 13F support
3. Implement CIK query mode: download all Form 4s for a CIK within date range
4. Add batch mode for processing multiple filings

**Library Enhancements:**
1. Add more test cases (see TESTING.md for recommendations)
2. Add footnote resolution helpers
3. CSV export functionality
4. Add validation for parsed data
