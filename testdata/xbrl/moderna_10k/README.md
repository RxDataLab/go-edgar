# Moderna 10-K Test Case

## Overview

This directory contains a real Moderna 10-K filing (FY2024) for testing XBRL parsing functionality.

## Files

- `input.htm` - Inline XBRL (iXBRL) 10-K filing (2.6 MB)
- `metadata.json` - Filing metadata (company, CIK, dates, URL)
- `README.md` - This file

## Filing Details

- **Company**: Moderna, Inc.
- **CIK**: 0001682852
- **Form**: 10-K (Annual Report)
- **Filing Date**: 2025-02-21
- **Report Period**: 2024-12-31 (FY2024)
- **Accession**: 0001682852-25-000022
- **Format**: Inline XBRL (iXBRL)
- **Source**: https://www.sec.gov/Archives/edgar/data/1682852/000168285225000022/mrna-20241231.htm

## Why Moderna?

Moderna is an ideal test case for biotech-focused XBRL parsing because:

1. **Pure biotech**: mRNA vaccine company with heavy R&D focus
2. **Commercial stage**: Has revenue from COVID-19 vaccines but still R&D intensive
3. **Clean financials**: Public company with well-structured XBRL filings
4. **Recent data**: FY2024 filing with latest US-GAAP taxonomy

## Expected Metrics to Extract

Based on biotech focus, this test should validate extraction of:

| Metric | XBRL Concept | Expected Approx. Value |
|--------|--------------|------------------------|
| Cash & Equivalents | us-gaap:CashAndCashEquivalentsAtCarryingValue | ~$7-9B |
| R&D Expense | us-gaap:ResearchAndDevelopmentExpense | ~$5-6B |
| G&A Expense | us-gaap:GeneralAndAdministrativeExpense | ~$1-2B |
| Revenue | us-gaap:Revenues | ~$6-8B |
| Net Income/Loss | us-gaap:NetIncomeLoss | Varies |
| Diluted Shares | us-gaap:WeightedAverageNumberOfDilutedSharesOutstanding | ~350-400M |

## Inline XBRL Format

This filing uses **inline XBRL (iXBRL)**, where XBRL facts are embedded in HTML with the `ix:` namespace:

```html
<ix:nonFraction contextRef="c-4" name="us-gaap:Cash" unitRef="usd" decimals="-6">
  1234
</ix:nonFraction>
```

Key namespaces:
- `ix` - Inline XBRL tags (http://www.xbrl.org/2013/inlineXBRL)
- `xbrli` - XBRL instance elements (contexts, units)
- `us-gaap` - US GAAP taxonomy
- `mrna` - Company-specific extensions
- `dei` - Document and Entity Information

## Testing Strategy

1. **Parse iXBRL HTML** - Extract facts from `ix:nonFraction`, `ix:nonNumeric` tags
2. **Resolve contexts** - Map contextRef to period (instant vs duration)
3. **Apply concept mappings** - Standardize US-GAAP concepts to common labels
4. **Validate metrics** - Ensure extracted values match expected ranges
5. **Test query API** - Verify filtering by concept, period, etc.

## Notes

- This is inline XBRL embedded in HTML, not standalone XBRL XML
- The parser needs to handle the `ix:` namespace and extract embedded facts
- Contexts and units are defined in `<ix:resources>` within the HTML
- Financial data appears in both the HTML (for human reading) and as iXBRL tags (for machine parsing)
