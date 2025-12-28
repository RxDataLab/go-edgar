package edgar

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
)

// Form4 represents an SEC Form 4 insider trading filing
type Form4 struct {
	XMLName            xml.Name            `xml:"ownershipDocument"`
	SchemaVersion      string              `xml:"schemaVersion"`
	DocumentType       string              `xml:"documentType"`
	PeriodOfReport     string              `xml:"periodOfReport"`
	Aff10b5One         bool                `xml:"aff10b5One"` // 10b5-1 trading plan indicator
	Issuer             Issuer              `xml:"issuer"`
	ReportingOwners    []ReportingOwner    `xml:"reportingOwner"`
	NonDerivativeTable *NonDerivativeTable `xml:"nonDerivativeTable"`
	DerivativeTable    *DerivativeTable    `xml:"derivativeTable"`
	Footnotes          []Footnote          `xml:"footnotes>footnote"`
	Signatures         []Signature         `xml:"ownerSignature"`
	Remarks            string              `xml:"remarks"`
}

// Issuer represents the company whose stock is being traded
type Issuer struct {
	CIK           string `xml:"issuerCik"`
	Name          string `xml:"issuerName"`
	TradingSymbol string `xml:"issuerTradingSymbol"`
}

// ReportingOwner represents an insider filing the Form 4
type ReportingOwner struct {
	ID           OwnerID      `xml:"reportingOwnerId"`
	Address      OwnerAddress `xml:"reportingOwnerAddress"`
	Relationship Relationship `xml:"reportingOwnerRelationship"`
}

type OwnerID struct {
	CIK  string `xml:"rptOwnerCik"`
	Name string `xml:"rptOwnerName"`
}

type OwnerAddress struct {
	Street1 string `xml:"rptOwnerStreet1"`
	Street2 string `xml:"rptOwnerStreet2"`
	City    string `xml:"rptOwnerCity"`
	State   string `xml:"rptOwnerState"`
	ZipCode string `xml:"rptOwnerZipCode"`
}

type Relationship struct {
	IsDirector        bool   `xml:"isDirector"`
	IsOfficer         bool   `xml:"isOfficer"`
	IsTenPercentOwner bool   `xml:"isTenPercentOwner"`
	IsOther           bool   `xml:"isOther"`
	OfficerTitle      string `xml:"officerTitle"`
}

// NonDerivativeTable contains common stock transactions
type NonDerivativeTable struct {
	Transactions []NonDerivativeTransaction `xml:"nonDerivativeTransaction"`
	Holdings     []NonDerivativeHolding     `xml:"nonDerivativeHolding"`
}

// NonDerivativeTransaction represents a stock purchase, sale, or grant
type NonDerivativeTransaction struct {
	SecurityTitle   string                 `xml:"securityTitle>value"`
	TransactionDate string                 `xml:"transactionDate>value"`
	Coding          TransactionCoding      `xml:"transactionCoding"`
	Amounts         TransactionAmounts     `xml:"transactionAmounts"`
	PostTransaction PostTransactionAmounts `xml:"postTransactionAmounts"`
	OwnershipNature OwnershipNature        `xml:"ownershipNature"`
}

type TransactionCoding struct {
	FormType           string     `xml:"transactionFormType"`
	Code               string     `xml:"transactionCode"`
	EquitySwapInvolved bool       `xml:"equitySwapInvolved"`
	FootnoteID         FootnoteID `xml:"footnoteId"`
}

type TransactionAmounts struct {
	Shares           Value  `xml:"transactionShares"`
	PricePerShare    Value  `xml:"transactionPricePerShare"`
	AcquiredDisposed string `xml:"transactionAcquiredDisposedCode>value"`
}

type PostTransactionAmounts struct {
	SharesOwnedFollowing Value `xml:"sharesOwnedFollowingTransaction"`
}

type OwnershipNature struct {
	DirectOrIndirect  string `xml:"directOrIndirectOwnership>value"`
	NatureOfOwnership string `xml:"natureOfOwnership>value"`
}

type Value struct {
	Value      string     `xml:"value"`
	FootnoteID FootnoteID `xml:"footnoteId"`
}

type FootnoteID struct {
	ID string `xml:"id,attr"`
}

// Footnote returns the footnote ID as a string (for convenience)
func (v Value) Footnote() string {
	return v.FootnoteID.ID
}

// Float64 returns the value as float64, handling empty values and footnote refs
func (v Value) Float64() (float64, error) {
	if v.Value == "" {
		return 0, fmt.Errorf("empty value")
	}
	return strconv.ParseFloat(v.Value, 64)
}

// Int returns the value as int
func (v Value) Int() (int, error) {
	if v.Value == "" {
		return 0, fmt.Errorf("empty value")
	}
	return strconv.Atoi(v.Value)
}

// DerivativeTable contains option/derivative transactions
type DerivativeTable struct {
	Transactions []DerivativeTransaction `xml:"derivativeTransaction"`
	Holdings     []DerivativeHolding     `xml:"derivativeHolding"`
}

type DerivativeTransaction struct {
	SecurityTitle             string                 `xml:"securityTitle>value"`
	ConversionOrExercisePrice Value                  `xml:"conversionOrExercisePrice"`
	TransactionDate           string                 `xml:"transactionDate>value"`
	Coding                    TransactionCoding      `xml:"transactionCoding"`
	Amounts                   TransactionAmounts     `xml:"transactionAmounts"`
	ExerciseDate              Value                  `xml:"exerciseDate"`
	ExpirationDate            Value                  `xml:"expirationDate"`
	UnderlyingSecurity        UnderlyingSecurity     `xml:"underlyingSecurity"`
	PostTransaction           PostTransactionAmounts `xml:"postTransactionAmounts"`
	OwnershipNature           OwnershipNature        `xml:"ownershipNature"`
}

type DerivativeHolding struct {
	SecurityTitle             string                 `xml:"securityTitle>value"`
	ConversionOrExercisePrice Value                  `xml:"conversionOrExercisePrice"`
	ExerciseDate              Value                  `xml:"exerciseDate"`
	ExpirationDate            Value                  `xml:"expirationDate"`
	UnderlyingSecurity        UnderlyingSecurity     `xml:"underlyingSecurity"`
	PostTransaction           PostTransactionAmounts `xml:"postTransactionAmounts"`
	OwnershipNature           OwnershipNature        `xml:"ownershipNature"`
}

type NonDerivativeHolding struct {
	SecurityTitle string `xml:"securityTitle>value"`
	// Add more fields as needed
}

// UnderlyingSecurity represents the security underlying a derivative
type UnderlyingSecurity struct {
	SecurityTitle Value `xml:"underlyingSecurityTitle"`
	Shares        Value `xml:"underlyingSecurityShares"`
}

type Footnote struct {
	ID   string `xml:"id,attr"`
	Text string `xml:",chardata"`
}

type Signature struct {
	Name string `xml:"signatureName"`
	Date string `xml:"signatureDate"`
}

// Parse unmarshals Form 4 XML into a Form4 struct
func Parse(data []byte) (*Form4, error) {
	var form4 Form4
	if err := xml.Unmarshal(data, &form4); err != nil {
		return nil, err
	}
	return &form4, nil
}

// TransactionCodeDescription returns human-readable transaction code
func TransactionCodeDescription(code string) string {
	descriptions := map[string]string{
		"P": "Open Market Purchase",
		"S": "Open Market Sale",
		"A": "Grant, Award or Other Acquisition",
		"D": "Disposition to the Issuer",
		"F": "Payment of Exercise Price or Tax Liability",
		"G": "Gift",
		"M": "Exercise or Conversion of Derivative Security",
		"C": "Conversion of Derivative Security",
		"E": "Expiration of Short Derivative Position",
		"H": "Expiration of Long Derivative Position",
		"I": "Discretionary Transaction",
		"O": "Exercise of Out-of-the-Money Derivative Security",
		"U": "Disposition Pursuant to a Tender",
		"X": "Exercise of In-the-Money or At-the-Money Derivative Security",
		"Z": "Deposit into or Withdrawal from Voting Trust",
	}
	return descriptions[code]
}

// GetMarketTrades returns only open market purchases and sales
func (f *Form4) GetMarketTrades() []NonDerivativeTransaction {
	if f.NonDerivativeTable == nil {
		return nil
	}

	var trades []NonDerivativeTransaction
	for _, txn := range f.NonDerivativeTable.Transactions {
		if txn.Coding.Code == "P" || txn.Coding.Code == "S" {
			trades = append(trades, txn)
		}
	}
	return trades
}

// GetPurchases returns only open market purchases
func (f *Form4) GetPurchases() []NonDerivativeTransaction {
	var purchases []NonDerivativeTransaction
	for _, txn := range f.GetMarketTrades() {
		if txn.Coding.Code == "P" {
			purchases = append(purchases, txn)
		}
	}
	return purchases
}

// GetSales returns only open market sales
func (f *Form4) GetSales() []NonDerivativeTransaction {
	var sales []NonDerivativeTransaction
	for _, txn := range f.GetMarketTrades() {
		if txn.Coding.Code == "S" {
			sales = append(sales, txn)
		}
	}
	return sales
}

// Is10b51Plan returns true if the form indicates a 10b5-1 trading plan
// Checks both the XML flag (aff10b5One) and footnote text
func (f *Form4) Is10b51Plan() bool {
	// Check XML flag
	if f.Aff10b5One {
		return true
	}

	// Check footnotes for 10b5-1 mentions
	pattern := regexp.MustCompile(`(?i)10b5-1\s+trading\s+plan`)
	for _, fn := range f.Footnotes {
		if pattern.MatchString(fn.Text) {
			return true
		}
	}

	return false
}

// Get10b51AdoptionDate extracts the adoption date from footnotes
// Pattern: "10b5-1 trading plan adopted.*on (Month DD, YYYY)"
// Returns empty string if not found
func (f *Form4) Get10b51AdoptionDate() string {
	pattern := regexp.MustCompile(`(?i)10b5-1\s+trading\s+plan\s+adopted.*on\s+([A-Za-z]+\s+\d{1,2},\s+\d{4})`)

	for _, fn := range f.Footnotes {
		matches := pattern.FindStringSubmatch(fn.Text)
		if len(matches) > 1 {
			return matches[1] // Return captured date string
		}
	}

	return ""
}

// IsUnder10b51 checks if this transaction references a 10b5-1 footnote
func (t *NonDerivativeTransaction) IsUnder10b51(form *Form4) bool {
	// Check if transaction coding has footnote ref
	if t.Coding.FootnoteID.ID != "" {
		// Look up footnote and check for 10b5-1 mention
		for _, fn := range form.Footnotes {
			if fn.ID == t.Coding.FootnoteID.ID {
				pattern := regexp.MustCompile(`(?i)10b5-1\s+trading\s+plan`)
				return pattern.MatchString(fn.Text)
			}
		}
	}
	return false
}
