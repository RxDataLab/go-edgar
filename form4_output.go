package edgar

// Form4Output represents the simplified JSON output structure
type Form4Output struct {
	FormType        string                        `json:"formType"`
	SchemaVersion   string                        `json:"schemaVersion"`
	PeriodOfReport  string                        `json:"periodOfReport"`
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
	Is10b51Plan           bool     `json:"is10b51Plan"`                     // Per-transaction 10b5-1 indicator
	Plan10b51AdoptionDate string   `json:"plan10b51AdoptionDate,omitempty"` // ISO-8601 date (YYYY-MM-DD), empty if not 10b5-1 or date unknown
	Footnotes             []string `json:"footnotes"`                       // Array of footnote IDs
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
	Is10b51Plan           bool     `json:"is10b51Plan"`                     // Per-transaction 10b5-1 indicator
	Plan10b51AdoptionDate string   `json:"plan10b51AdoptionDate,omitempty"` // ISO-8601 date (YYYY-MM-DD), empty if not 10b5-1 or date unknown
	Footnotes             []string `json:"footnotes"`                       // Array of footnote IDs
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

// ToOutput converts a Form4 to the simplified output structure
func (f *Form4) ToOutput() *Form4Output {
	// Parse footnotes once to identify 10b5-1 plans and adoption dates
	tenb51Map := f.Parse10b51Footnotes()

	out := &Form4Output{
		FormType:        f.DocumentType,
		SchemaVersion:   f.SchemaVersion,
		PeriodOfReport:  f.PeriodOfReport,
		Has10b51Plan:    f.Is10b51Plan(),
		Issuer:          convertIssuer(f.Issuer),
		ReportingOwners: convertReportingOwners(f.ReportingOwners),
		Footnotes:       convertFootnotes(f.Footnotes),
		Signatures:      convertSignatures(f.Signatures),
	}

	// Convert non-derivative transactions
	if f.NonDerivativeTable != nil {
		for _, txn := range f.NonDerivativeTable.Transactions {
			out.Transactions = append(out.Transactions, convertNonDerivTransaction(txn, tenb51Map))
		}
		for _, holding := range f.NonDerivativeTable.Holdings {
			out.Holdings = append(out.Holdings, convertNonDerivHolding(holding))
		}
	}

	// Convert derivative transactions
	if f.DerivativeTable != nil {
		for _, txn := range f.DerivativeTable.Transactions {
			out.Derivatives = append(out.Derivatives, convertDerivTransaction(txn, tenb51Map))
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

func convertNonDerivTransaction(txn NonDerivativeTransaction, tenb51Map map[string]string) NonDerivativeTransactionOut {
	// Collect all footnote IDs
	footnotes := collectFootnotes(
		txn.Coding.FootnoteID.ID,
		txn.Amounts.Shares.FootnoteID.ID,
		txn.Amounts.PricePerShare.FootnoteID.ID,
		txn.PostTransaction.SharesOwnedFollowing.FootnoteID.ID,
	)

	// Check if any footnote indicates 10b5-1 plan
	is10b51, adoptionDate := check10b51Plan(footnotes, tenb51Map)

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

func convertDerivTransaction(txn DerivativeTransaction, tenb51Map map[string]string) DerivativeTransactionOut {
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
	is10b51, adoptionDate := check10b51Plan(footnotes, tenb51Map)

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

func convertFootnotes(footnotes []Footnote) []FootnoteOutput {
	var out []FootnoteOutput
	for _, fn := range footnotes {
		out = append(out, FootnoteOutput{
			ID:   fn.ID,
			Text: fn.Text,
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

// check10b51Plan checks if any of the transaction's footnotes indicate a 10b5-1 plan
// Returns: (is10b51Plan bool, adoptionDate string)
// If multiple footnotes reference 10b5-1, returns the first non-empty adoption date found
func check10b51Plan(footnoteIDs []string, tenb51Map map[string]string) (bool, string) {
	for _, fnID := range footnoteIDs {
		if adoptionDate, exists := tenb51Map[fnID]; exists {
			// This footnote indicates 10b5-1 plan
			return true, adoptionDate // Returns date (may be empty string if not found in footnote)
		}
	}
	return false, ""
}
