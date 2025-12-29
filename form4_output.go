package edgar

// Form4Output represents the simplified JSON output structure
type Form4Output struct {
	Metadata        FormMetadata                  `json:"metadata"`
	SchemaVersion   string                        `json:"schemaVersion"`
	Has10b51Plan    bool                          `json:"has10b51Plan"` // Document-level indicator
	Issuer          IssuerOutput                  `json:"issuer"`
	ReportingOwners []ReportingOwnerOutput        `json:"reportingOwners"`
	Transactions    []NonDerivativeTransactionOut `json:"transactions"`
	Derivatives     []DerivativeTransactionOut    `json:"derivatives"`
	Holdings        []NonDerivativeHoldingOut     `json:"holdings,omitempty"`
	DerivHoldings   []DerivativeHoldingOut        `json:"derivativeHoldings,omitempty"`
	Footnotes       []FootnoteOutput              `json:"footnotes"`
	Signatures      []SignatureOutput             `json:"signatures"`
}

// FormMetadata contains metadata about the filing
type FormMetadata struct {
	CIK             string `json:"cik"`
	AccessionNumber string `json:"accessionNumber"`
	FormType        string `json:"formType"`
	PeriodOfReport  string `json:"periodOfReport"`
	FilingDate      string `json:"filingDate"` // From SEC index, empty if not available
	ReportDate      string `json:"reportDate"` // From SEC index, empty if not available
	Source          string `json:"source"`     // URL or file path
}

type IssuerOutput struct {
	CIK    string `json:"cik"`
	Name   string `json:"name"`
	Ticker string `json:"ticker"`
}

type ReportingOwnerOutput struct {
	CIK          string          `json:"cik"`
	Name         string          `json:"name"`
	Address      AddressOutput   `json:"address"`
	Relationship RelationshipOut `json:"relationship"`
}

type AddressOutput struct {
	Street1 string `json:"street1,omitempty"`
	Street2 string `json:"street2,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	ZipCode string `json:"zipCode,omitempty"`
}

type RelationshipOut struct {
	IsDirector        bool   `json:"isDirector"`
	IsOfficer         bool   `json:"isOfficer"`
	IsTenPercentOwner bool   `json:"isTenPercentOwner"`
	IsOther           bool   `json:"isOther"`
	OfficerTitle      string `json:"officerTitle,omitempty"`
}

// NonDerivativeTransactionOut represents a single transaction row (table-like)
type NonDerivativeTransactionOut struct {
	SecurityTitle         string   `json:"securityTitle"`
	TransactionDate       string   `json:"transactionDate"`
	TransactionCode       string   `json:"transactionCode"`
	Shares                *float64 `json:"shares"`               // Nullable for empty values
	PricePerShare         *float64 `json:"pricePerShare"`        // Nullable for empty values
	AcquiredDisposed      string   `json:"acquiredDisposed"`     // "A" or "D"
	SharesOwnedFollowing  *float64 `json:"sharesOwnedFollowing"` // Nullable
	DirectIndirect        string   `json:"directIndirect"`       // "D" or "I"
	NatureOfOwnership     string   `json:"natureOfOwnership,omitempty"`
	EquitySwapInvolved    bool     `json:"equitySwapInvolved"`
	Is10b51Plan           bool     `json:"is10b51Plan"`           // Per-transaction 10b5-1 indicator (always present)
	Plan10b51AdoptionDate *string  `json:"plan10b51AdoptionDate"` // ISO-8601 date (YYYY-MM-DD), null if not 10b5-1 or date unknown (always present)
	Footnotes             []string `json:"footnotes"`             // Array of footnote IDs
}

// DerivativeTransactionOut represents a derivative transaction row
type DerivativeTransactionOut struct {
	SecurityTitle         string   `json:"securityTitle"`
	TransactionDate       string   `json:"transactionDate"`
	TransactionCode       string   `json:"transactionCode"`
	Shares                *float64 `json:"shares"`
	PricePerShare         *float64 `json:"pricePerShare"`
	AcquiredDisposed      string   `json:"acquiredDisposed"`
	ExercisePrice         *float64 `json:"exercisePrice,omitempty"`
	ExerciseDate          string   `json:"exerciseDate,omitempty"`
	ExpirationDate        string   `json:"expirationDate,omitempty"`
	UnderlyingTitle       string   `json:"underlyingTitle,omitempty"`
	UnderlyingShares      *float64 `json:"underlyingShares,omitempty"`
	SharesOwnedFollowing  *float64 `json:"sharesOwnedFollowing"`
	DirectIndirect        string   `json:"directIndirect"`
	NatureOfOwnership     string   `json:"natureOfOwnership,omitempty"`
	EquitySwapInvolved    bool     `json:"equitySwapInvolved"`
	Is10b51Plan           bool     `json:"is10b51Plan"`           // Per-transaction 10b5-1 indicator (always present)
	Plan10b51AdoptionDate *string  `json:"plan10b51AdoptionDate"` // ISO-8601 date (YYYY-MM-DD), null if not 10b5-1 or date unknown (always present)
	Footnotes             []string `json:"footnotes"`             // Array of footnote IDs
}

// NonDerivativeHoldingOut represents a holding row
type NonDerivativeHoldingOut struct {
	SecurityTitle        string   `json:"securityTitle"`
	SharesOwnedFollowing *float64 `json:"sharesOwnedFollowing"`
	DirectIndirect       string   `json:"directIndirect"`
	NatureOfOwnership    string   `json:"natureOfOwnership,omitempty"`
	Footnotes            []string `json:"footnotes"`
}

// DerivativeHoldingOut represents a derivative holding row
type DerivativeHoldingOut struct {
	SecurityTitle        string   `json:"securityTitle"`
	ExercisePrice        *float64 `json:"exercisePrice,omitempty"`
	ExerciseDate         string   `json:"exerciseDate,omitempty"`
	ExpirationDate       string   `json:"expirationDate,omitempty"`
	UnderlyingTitle      string   `json:"underlyingTitle,omitempty"`
	UnderlyingShares     *float64 `json:"underlyingShares,omitempty"`
	SharesOwnedFollowing *float64 `json:"sharesOwnedFollowing"`
	DirectIndirect       string   `json:"directIndirect"`
	NatureOfOwnership    string   `json:"natureOfOwnership,omitempty"`
	Footnotes            []string `json:"footnotes"`
}

type FootnoteOutput struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type SignatureOutput struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

// SetSource sets the source field in the metadata (URL or file path)
func (f *Form4Output) SetSource(source string) {
	f.Metadata.Source = source
}

// SetFilingMetadata sets filing metadata fields from external sources (e.g., SEC index)
func (f *Form4Output) SetFilingMetadata(accessionNumber, filingDate, reportDate string) {
	if accessionNumber != "" {
		f.Metadata.AccessionNumber = accessionNumber
	}
	if filingDate != "" {
		f.Metadata.FilingDate = filingDate
	}
	if reportDate != "" {
		f.Metadata.ReportDate = reportDate
	}
}

// ToOutput converts a Form4 to the simplified output structure
func (f *Form4) ToOutput() *Form4Output {
	// Parse footnotes and remarks once to identify 10b5-1 plans and adoption dates
	//
	// Priority order for 10b5-1 detection:
	// 1. Transaction-specific footnotes (highest priority)
	// 2. Remarks field (fallback when aff10b5One=true but NO footnotes mention 10b5-1)
	// 3. Not a 10b5-1 transaction
	//
	// The map contains footnote IDs -> adoption dates (ISO format)
	// Special key "__REMARKS__" is used when remarks contains 10b5-1 info
	tenb51Map := f.Parse10b51Footnotes()

	// Check if we should use remarks as global fallback
	// Only use remarks if: aff10b5One=true AND no footnotes mention 10b5-1
	// This handles cases like Becton Dickinson where 10b5-1 info is ONLY in remarks
	has10b51Footnotes := false
	for k := range tenb51Map {
		if k != "__REMARKS__" {
			has10b51Footnotes = true
			break
		}
	}
	useRemarksGlobal := f.Aff10b5One && !has10b51Footnotes && tenb51Map["__REMARKS__"] != ""

	out := &Form4Output{
		Metadata: FormMetadata{
			CIK:             f.Issuer.CIK,
			AccessionNumber: "", // To be filled by caller if available
			FormType:        f.DocumentType,
			PeriodOfReport:  f.PeriodOfReport,
			FilingDate:      "", // To be filled by caller if available
			ReportDate:      "", // To be filled by caller if available
			Source:          "", // To be filled by caller if available
		},
		SchemaVersion:   f.SchemaVersion,
		Has10b51Plan:    f.Is10b51Plan(),
		Issuer:          convertIssuer(f.Issuer),
		ReportingOwners: convertReportingOwners(f.ReportingOwners),
		Footnotes:       convertFootnotes(f.Footnotes, f.Remarks),
		Signatures:      convertSignatures(f.Signatures),
	}

	// Convert non-derivative transactions
	if f.NonDerivativeTable != nil {
		for _, txn := range f.NonDerivativeTable.Transactions {
			out.Transactions = append(out.Transactions, convertNonDerivTransaction(txn, tenb51Map, useRemarksGlobal))
		}
		for _, holding := range f.NonDerivativeTable.Holdings {
			out.Holdings = append(out.Holdings, convertNonDerivHolding(holding))
		}
	}

	// Convert derivative transactions
	if f.DerivativeTable != nil {
		for _, txn := range f.DerivativeTable.Transactions {
			out.Derivatives = append(out.Derivatives, convertDerivTransaction(txn, tenb51Map, useRemarksGlobal))
		}
		for _, holding := range f.DerivativeTable.Holdings {
			out.DerivHoldings = append(out.DerivHoldings, convertDerivHolding(holding))
		}
	}

	return out
}

func convertIssuer(i Issuer) IssuerOutput {
	return IssuerOutput{
		CIK:    i.CIK,
		Name:   i.Name,
		Ticker: i.TradingSymbol,
	}
}

func convertReportingOwners(owners []ReportingOwner) []ReportingOwnerOutput {
	var out []ReportingOwnerOutput
	for _, owner := range owners {
		out = append(out, ReportingOwnerOutput{
			CIK:  owner.ID.CIK,
			Name: owner.ID.Name,
			Address: AddressOutput{
				Street1: owner.Address.Street1,
				Street2: owner.Address.Street2,
				City:    owner.Address.City,
				State:   owner.Address.State,
				ZipCode: owner.Address.ZipCode,
			},
			Relationship: RelationshipOut{
				IsDirector:        owner.Relationship.IsDirector,
				IsOfficer:         owner.Relationship.IsOfficer,
				IsTenPercentOwner: owner.Relationship.IsTenPercentOwner,
				IsOther:           owner.Relationship.IsOther,
				OfficerTitle:      owner.Relationship.OfficerTitle,
			},
		})
	}
	return out
}

func convertNonDerivTransaction(txn NonDerivativeTransaction, tenb51Map map[string]string, useRemarksGlobal bool) NonDerivativeTransactionOut {
	// Collect all footnote IDs
	footnotes := collectFootnotes(
		txn.Coding.FootnoteID.ID,
		txn.Amounts.Shares.FootnoteID.ID,
		txn.Amounts.PricePerShare.FootnoteID.ID,
		txn.PostTransaction.SharesOwnedFollowing.FootnoteID.ID,
	)

	// Check if any footnote indicates 10b5-1 plan
	is10b51, adoptionDate := check10b51Plan(footnotes, tenb51Map, useRemarksGlobal)

	return NonDerivativeTransactionOut{
		SecurityTitle:         txn.SecurityTitle,
		TransactionDate:       txn.TransactionDate,
		TransactionCode:       txn.Coding.Code,
		Shares:                toFloat64Ptr(txn.Amounts.Shares),
		PricePerShare:         toFloat64Ptr(txn.Amounts.PricePerShare),
		AcquiredDisposed:      txn.Amounts.AcquiredDisposed,
		SharesOwnedFollowing:  toFloat64Ptr(txn.PostTransaction.SharesOwnedFollowing),
		DirectIndirect:        txn.OwnershipNature.DirectOrIndirect,
		NatureOfOwnership:     txn.OwnershipNature.NatureOfOwnership,
		EquitySwapInvolved:    txn.Coding.EquitySwapInvolved,
		Is10b51Plan:           is10b51,
		Plan10b51AdoptionDate: adoptionDate,
		Footnotes:             footnotes,
	}
}

func convertDerivTransaction(txn DerivativeTransaction, tenb51Map map[string]string, useRemarksGlobal bool) DerivativeTransactionOut {
	footnotes := collectFootnotes(
		txn.Coding.FootnoteID.ID,
		txn.Amounts.Shares.FootnoteID.ID,
		txn.Amounts.PricePerShare.FootnoteID.ID,
		txn.ConversionOrExercisePrice.FootnoteID.ID,
		txn.ExerciseDate.FootnoteID.ID,
		txn.ExpirationDate.FootnoteID.ID,
		txn.UnderlyingSecurity.SecurityTitle.FootnoteID.ID,
		txn.UnderlyingSecurity.Shares.FootnoteID.ID,
		txn.PostTransaction.SharesOwnedFollowing.FootnoteID.ID,
	)

	// Check if any footnote indicates 10b5-1 plan
	is10b51, adoptionDate := check10b51Plan(footnotes, tenb51Map, useRemarksGlobal)

	return DerivativeTransactionOut{
		SecurityTitle:         txn.SecurityTitle,
		TransactionDate:       txn.TransactionDate,
		TransactionCode:       txn.Coding.Code,
		Shares:                toFloat64Ptr(txn.Amounts.Shares),
		PricePerShare:         toFloat64Ptr(txn.Amounts.PricePerShare),
		AcquiredDisposed:      txn.Amounts.AcquiredDisposed,
		ExercisePrice:         toFloat64Ptr(txn.ConversionOrExercisePrice),
		ExerciseDate:          txn.ExerciseDate.Value,
		ExpirationDate:        txn.ExpirationDate.Value,
		UnderlyingTitle:       txn.UnderlyingSecurity.SecurityTitle.Value,
		UnderlyingShares:      toFloat64Ptr(txn.UnderlyingSecurity.Shares),
		SharesOwnedFollowing:  toFloat64Ptr(txn.PostTransaction.SharesOwnedFollowing),
		DirectIndirect:        txn.OwnershipNature.DirectOrIndirect,
		NatureOfOwnership:     txn.OwnershipNature.NatureOfOwnership,
		EquitySwapInvolved:    txn.Coding.EquitySwapInvolved,
		Is10b51Plan:           is10b51,
		Plan10b51AdoptionDate: adoptionDate,
		Footnotes:             footnotes,
	}
}

func convertNonDerivHolding(holding NonDerivativeHolding) NonDerivativeHoldingOut {
	// TODO: Add fields when we have test data with holdings
	return NonDerivativeHoldingOut{
		SecurityTitle: holding.SecurityTitle,
		Footnotes:     []string{},
	}
}

func convertDerivHolding(holding DerivativeHolding) DerivativeHoldingOut {
	footnotes := collectFootnotes(
		holding.ConversionOrExercisePrice.FootnoteID.ID,
		holding.ExerciseDate.FootnoteID.ID,
		holding.ExpirationDate.FootnoteID.ID,
		holding.UnderlyingSecurity.SecurityTitle.FootnoteID.ID,
		holding.UnderlyingSecurity.Shares.FootnoteID.ID,
		holding.PostTransaction.SharesOwnedFollowing.FootnoteID.ID,
	)

	return DerivativeHoldingOut{
		SecurityTitle:        holding.SecurityTitle,
		ExercisePrice:        toFloat64Ptr(holding.ConversionOrExercisePrice),
		ExerciseDate:         holding.ExerciseDate.Value,
		ExpirationDate:       holding.ExpirationDate.Value,
		UnderlyingTitle:      holding.UnderlyingSecurity.SecurityTitle.Value,
		UnderlyingShares:     toFloat64Ptr(holding.UnderlyingSecurity.Shares),
		SharesOwnedFollowing: toFloat64Ptr(holding.PostTransaction.SharesOwnedFollowing),
		DirectIndirect:       holding.OwnershipNature.DirectOrIndirect,
		NatureOfOwnership:    holding.OwnershipNature.NatureOfOwnership,
		Footnotes:            footnotes,
	}
}

func convertFootnotes(footnotes []Footnote, remarks string) []FootnoteOutput {
	var out []FootnoteOutput
	for _, fn := range footnotes {
		out = append(out, FootnoteOutput{
			ID:   fn.ID,
			Text: fn.Text,
		})
	}

	// Include remarks as a footnote with ID "REMARKS" if non-empty
	if remarks != "" {
		out = append(out, FootnoteOutput{
			ID:   "REMARKS",
			Text: remarks,
		})
	}

	return out
}

func convertSignatures(sigs []Signature) []SignatureOutput {
	var out []SignatureOutput
	for _, sig := range sigs {
		out = append(out, SignatureOutput{
			Name: sig.Name,
			Date: sig.Date,
		})
	}
	return out
}

// toFloat64Ptr converts a Value to *float64, returning nil if parsing fails
func toFloat64Ptr(v Value) *float64 {
	f, err := v.Float64()
	if err != nil {
		return nil
	}
	return &f
}

// collectFootnotes returns a deduplicated list of footnote IDs (excluding empty strings)
func collectFootnotes(ids ...string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}

// check10b51Plan checks if a transaction is part of a 10b5-1 trading plan
//
// Detection priority (strictest to least strict):
//  1. Check transaction-specific footnotes for 10b5-1 mentions (HIGHEST PRIORITY)
//  2. If useRemarksGlobal=true, apply remarks-based 10b5-1 to ALL transactions (FALLBACK)
//  3. Not a 10b5-1 transaction
//
// useRemarksGlobal should only be true when:
//   - aff10b5One XML flag is true (form declares 10b5-1 plan)
//   - AND no footnotes mention 10b5-1 (remarks is the only source)
//
// Returns: (is10b51Plan bool, adoptionDate *string)
//   - adoptionDate is nil if plan exists but no date found
//   - adoptionDate is non-nil pointer to ISO date string if date found
func check10b51Plan(footnoteIDs []string, tenb51Map map[string]string, useRemarksGlobal bool) (bool, *string) {
	// Priority 1: Check transaction-specific footnotes
	// If a footnote explicitly mentions 10b5-1, that takes precedence over everything
	for _, fnID := range footnoteIDs {
		if adoptionDate, exists := tenb51Map[fnID]; exists {
			// This footnote indicates 10b5-1 plan
			if adoptionDate != "" {
				return true, &adoptionDate
			}
			return true, nil // 10b5-1 plan but no date found
		}
	}

	// Priority 2: Check if remarks should apply globally
	// Only when aff10b5One=true AND no footnotes mention 10b5-1
	// This handles cases like Becton Dickinson where 10b5-1 info is ONLY in remarks
	if useRemarksGlobal {
		if adoptionDate, exists := tenb51Map["__REMARKS__"]; exists {
			if adoptionDate != "" {
				return true, &adoptionDate
			}
			return true, nil
		}
	}

	return false, nil
}
