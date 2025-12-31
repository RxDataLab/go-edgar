package edgar

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// FactQuery provides a fluent interface for querying XBRL facts
type FactQuery struct {
	xbrl          *XBRL
	facts         []Fact
	conceptFilter []string
	labelFilter   string
	periodFilter  string
	instantOnly   bool
	durationOnly  bool
}

// Query returns a new FactQuery for the XBRL document
func (x *XBRL) Query() *FactQuery {
	return &FactQuery{
		xbrl:  x,
		facts: x.Facts,
	}
}

// ByConcept filters facts by XBRL concept name (e.g., "us-gaap:Cash")
func (q *FactQuery) ByConcept(concepts ...string) *FactQuery {
	q.conceptFilter = concepts
	return q
}

// ByLabel filters facts by standardized label (e.g., "Cash and Cash Equivalents")
func (q *FactQuery) ByLabel(label string) *FactQuery {
	q.labelFilter = label
	return q
}

// ForPeriodEndingOn filters facts by period end date (YYYY-MM-DD)
func (q *FactQuery) ForPeriodEndingOn(date string) *FactQuery {
	q.periodFilter = date
	return q
}

// InstantOnly returns only instant facts (balance sheet items)
func (q *FactQuery) InstantOnly() *FactQuery {
	q.instantOnly = true
	return q
}

// DurationOnly returns only duration facts (income statement items)
func (q *FactQuery) DurationOnly() *FactQuery {
	q.durationOnly = true
	return q
}

// Get returns all matching facts
func (q *FactQuery) Get() []Fact {
	var results []Fact

	for _, fact := range q.facts {
		// Apply concept filter
		if len(q.conceptFilter) > 0 {
			matched := false
			for _, concept := range q.conceptFilter {
				if fact.Concept == concept || strings.Contains(fact.Concept, concept) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Apply label filter
		if q.labelFilter != "" && fact.StandardLabel != q.labelFilter {
			continue
		}

		// Apply period filter
		if q.periodFilter != "" {
			endDate, err := fact.GetEndDate()
			if err != nil {
				continue
			}
			if endDate.Format("2006-01-02") != q.periodFilter {
				continue
			}
		}

		// Apply instant/duration filters
		if q.instantOnly && !fact.IsInstant() {
			continue
		}
		if q.durationOnly && !fact.IsDuration() {
			continue
		}

		results = append(results, fact)
	}

	return results
}

// First returns the first matching fact, or error if none found
func (q *FactQuery) First() (*Fact, error) {
	results := q.Get()
	if len(results) == 0 {
		return nil, fmt.Errorf("no facts found")
	}
	return &results[0], nil
}

// MostRecent returns the fact with the most recent period end date
func (q *FactQuery) MostRecent() (*Fact, error) {
	results := q.Get()
	if len(results) == 0 {
		return nil, fmt.Errorf("no facts found")
	}

	// Sort by end date descending
	sort.Slice(results, func(i, j int) bool {
		dateI, errI := results[i].GetEndDate()
		dateJ, errJ := results[j].GetEndDate()
		if errI != nil || errJ != nil {
			return false
		}
		return dateI.After(dateJ)
	})

	return &results[0], nil
}

// Sum returns the sum of all matching numeric facts
func (q *FactQuery) Sum() (float64, error) {
	results := q.Get()
	if len(results) == 0 {
		return 0, fmt.Errorf("no facts found")
	}

	var sum float64
	for _, fact := range results {
		if val, err := fact.Float64(); err == nil {
			sum += val
		}
	}

	return sum, nil
}

// High-level financial metric helpers

// GetCashAndEquivalents returns the most recent cash and equivalents value
func (x *XBRL) GetCashAndEquivalents() (float64, error) {
	fact, err := x.Query().
		ByLabel("Cash and Cash Equivalents").
		InstantOnly().
		MostRecent()

	if err != nil {
		return 0, fmt.Errorf("cash and equivalents not found: %w", err)
	}

	return fact.Float64()
}

// GetResearchAndDevelopment returns R&D expense for the most recent period
func (x *XBRL) GetResearchAndDevelopment(period string) (float64, error) {
	query := x.Query().
		ByLabel("Research and Development Expense").
		DurationOnly()

	if period != "" {
		query = query.ForPeriodEndingOn(period)
	}

	fact, err := query.MostRecent()
	if err != nil {
		return 0, fmt.Errorf("R&D expense not found: %w", err)
	}

	return fact.Float64()
}

// GetGeneralAndAdministrative returns G&A expense for the most recent period
func (x *XBRL) GetGeneralAndAdministrative(period string) (float64, error) {
	query := x.Query().
		ByLabel("General and Administrative Expense").
		DurationOnly()

	if period != "" {
		query = query.ForPeriodEndingOn(period)
	}

	fact, err := query.MostRecent()
	if err != nil {
		return 0, fmt.Errorf("G&A expense not found: %w", err)
	}

	return fact.Float64()
}

// GetBurn returns the quarterly or annual burn rate (R&D + G&A)
// If period is empty, returns most recent period
func (x *XBRL) GetBurn(period string) (float64, error) {
	rd, errRD := x.GetResearchAndDevelopment(period)
	ga, errGA := x.GetGeneralAndAdministrative(period)

	// Handle cases where one or both are missing
	if errRD != nil && errGA != nil {
		return 0, fmt.Errorf("neither R&D nor G&A found for burn calculation")
	}

	burn := 0.0
	if errRD == nil {
		burn += rd
	}
	if errGA == nil {
		burn += ga
	}

	return burn, nil
}

// GetTotalDebt returns total debt (long-term + short-term) as of most recent balance sheet
func (x *XBRL) GetTotalDebt() (float64, error) {
	// Get long-term debt
	ltDebt := 0.0
	if fact, err := x.Query().ByLabel("Long-Term Debt").InstantOnly().MostRecent(); err == nil {
		if val, err := fact.Float64(); err == nil {
			ltDebt = val
		}
	}

	// Get short-term debt
	stDebt := 0.0
	if fact, err := x.Query().ByLabel("Short-Term Debt").InstantOnly().MostRecent(); err == nil {
		if val, err := fact.Float64(); err == nil {
			stDebt = val
		}
	}

	// If both are zero, debt might not be reported (or company is debt-free)
	if ltDebt == 0 && stDebt == 0 {
		return 0, fmt.Errorf("no debt found (company may be debt-free)")
	}

	return ltDebt + stDebt, nil
}

// GetDilutedShares returns diluted shares outstanding for the most recent period
func (x *XBRL) GetDilutedShares(period string) (float64, error) {
	query := x.Query().
		ByLabel("Shares Outstanding (Diluted)").
		DurationOnly()

	if period != "" {
		query = query.ForPeriodEndingOn(period)
	}

	fact, err := query.MostRecent()
	if err != nil {
		return 0, fmt.Errorf("diluted shares not found: %w", err)
	}

	return fact.Float64()
}

// GetRevenue returns total revenue for the most recent period
func (x *XBRL) GetRevenue(period string) (float64, error) {
	query := x.Query().
		ByLabel("Revenue").
		DurationOnly()

	if period != "" {
		query = query.ForPeriodEndingOn(period)
	}

	fact, err := query.MostRecent()
	if err != nil {
		// Many biotech companies have $0 revenue
		return 0, nil
	}

	return fact.Float64()
}

// GetFinancialSnapshot returns a snapshot of key financial metrics
type FinancialSnapshot struct {
	// Period information
	FiscalYearEnd string `json:"fiscalYearEnd"`        // Fiscal year end date (YYYY-MM-DD)
	FilingDate    string `json:"filingDate,omitempty"` // When filed with SEC
	FiscalPeriod  string `json:"fiscalPeriod"`         // "FY" for 10-K, "Q1/Q2/Q3/Q4" for 10-Q
	FormType      string `json:"formType,omitempty"`   // "10-K", "10-Q", etc.

	// Company information
	CompanyName string `json:"companyName,omitempty"`
	CIK         string `json:"cik,omitempty"`

	// Validation
	MissingRequiredFields []string `json:"missingRequiredFields,omitempty"` // Required GAAP fields that are missing

	// Balance Sheet - Assets (instant, as of fiscal year end)
	Cash                   float64 `json:"cash"`
	AccountsReceivable     float64 `json:"accountsReceivable"`
	Inventory              float64 `json:"inventory"`
	PrepaidExpenses        float64 `json:"prepaidExpenses"`
	PropertyPlantEquipment float64 `json:"propertyPlantEquipment"`
	IntangibleAssets       float64 `json:"intangibleAssets"`
	Goodwill               float64 `json:"goodwill"`
	TotalAssets            float64 `json:"totalAssets"`

	// Balance Sheet - Liabilities (instant, as of fiscal year end)
	ShortTermDebt      float64 `json:"shortTermDebt"`
	LongTermDebt       float64 `json:"longTermDebt"`
	TotalDebt          float64 `json:"totalDebt"` // Short-term + Long-term
	AccountsPayable    float64 `json:"accountsPayable"`
	AccruedLiabilities float64 `json:"accruedLiabilities"`
	DeferredRevenue    float64 `json:"deferredRevenue"`
	TotalLiabilities   float64 `json:"totalLiabilities"`

	// Balance Sheet - Equity (instant, as of fiscal year end)
	StockholdersEquity           float64 `json:"stockholdersEquity"`
	AccumulatedDeficit           float64 `json:"accumulatedDeficit"`
	CommonStockSharesOutstanding float64 `json:"commonStockSharesOutstanding"`

	// Income Statement (duration, for the period)
	Revenue                 float64 `json:"revenue"`
	CostOfRevenue           float64 `json:"costOfRevenue"`
	GrossProfit             float64 `json:"grossProfit"`
	RDExpense               float64 `json:"rdExpense"`
	GAExpense               float64 `json:"gaExpense"`
	SellingMarketingExpense float64 `json:"sellingMarketingExpense"`
	TotalOperatingExpenses  float64 `json:"totalOperatingExpenses"`
	OperatingIncome         float64 `json:"operatingIncome"`
	InterestExpense         float64 `json:"interestExpense"`
	IncomeTaxExpense        float64 `json:"incomeTaxExpense"`
	NetIncome               float64 `json:"netIncome"`

	// Per Share Metrics (duration, for the period)
	BasicShares   float64 `json:"basicShares"`
	DilutedShares float64 `json:"dilutedShares"`
	EPSBasic      float64 `json:"epsBasic"`
	EPSDiluted    float64 `json:"epsDiluted"`

	// Cash Flow Statement (duration, for the period)
	CashFlowOperations  float64 `json:"cashFlowOperations"`
	CashFlowInvesting   float64 `json:"cashFlowInvesting"`
	CashFlowFinancing   float64 `json:"cashFlowFinancing"`
	CapitalExpenditures float64 `json:"capitalExpenditures"`

	// Non-Cash Items (duration, for the period)
	DepreciationAmortization float64 `json:"depreciationAmortization"`
	StockBasedCompensation   float64 `json:"stockBasedCompensation"`
}

// GetSnapshot returns a financial snapshot for the most recent period
func (x *XBRL) GetSnapshot() (*FinancialSnapshot, error) {
	snapshot := &FinancialSnapshot{}

	// Extract metadata from DEI (Document and Entity Information) facts
	extractMetadata(x, snapshot)

	// Find the fiscal year end date (latest annual/quarterly period)
	fiscalYearEnd := findFiscalYearEnd(x)
	if !fiscalYearEnd.IsZero() {
		snapshot.FiscalYearEnd = fiscalYearEnd.Format("2006-01-02")
	}

	// Helper function to get instant (balance sheet) metrics
	getInstant := func(label string) float64 {
		if fact, err := x.Query().ByLabel(label).InstantOnly().MostRecent(); err == nil {
			if val, err := fact.Float64(); err == nil {
				return val
			}
		}
		return 0
	}

	// Helper function to get duration (income/cash flow statement) metrics
	getDuration := func(label string) float64 {
		if fact, err := x.Query().ByLabel(label).DurationOnly().MostRecent(); err == nil {
			if val, err := fact.Float64(); err == nil {
				return val
			}
		}
		return 0
	}

	// Balance Sheet - Assets (instant)
	snapshot.Cash = getInstant("Cash and Cash Equivalents")
	snapshot.AccountsReceivable = getInstant("Accounts Receivable")
	snapshot.Inventory = getInstant("Inventory")
	snapshot.PrepaidExpenses = getInstant("Prepaid Expenses")
	snapshot.PropertyPlantEquipment = getInstant("Property Plant and Equipment")
	snapshot.IntangibleAssets = getInstant("Intangible Assets")
	snapshot.Goodwill = getInstant("Goodwill")
	snapshot.TotalAssets = getInstant("Total Assets")

	// Balance Sheet - Liabilities (instant)
	snapshot.ShortTermDebt = getInstant("Short-Term Debt")
	snapshot.LongTermDebt = getInstant("Long-Term Debt")
	snapshot.TotalDebt = snapshot.ShortTermDebt + snapshot.LongTermDebt
	snapshot.AccountsPayable = getInstant("Accounts Payable")
	snapshot.AccruedLiabilities = getInstant("Accrued Liabilities")
	snapshot.DeferredRevenue = getInstant("Deferred Revenue")
	snapshot.TotalLiabilities = getInstant("Total Liabilities")

	// Balance Sheet - Equity (instant)
	snapshot.StockholdersEquity = getInstant("Stockholders Equity")
	snapshot.AccumulatedDeficit = getInstant("Accumulated Deficit")
	snapshot.CommonStockSharesOutstanding = getInstant("Common Stock Shares Outstanding")

	// Income Statement (duration)
	snapshot.Revenue = getDuration("Revenue")
	snapshot.CostOfRevenue = getDuration("Cost of Revenue")
	snapshot.GrossProfit = getDuration("Gross Profit")
	snapshot.RDExpense = getDuration("Research and Development Expense")
	snapshot.GAExpense = getDuration("General and Administrative Expense")
	snapshot.SellingMarketingExpense = getDuration("Selling and Marketing Expense")
	snapshot.TotalOperatingExpenses = getDuration("Total Operating Expenses")
	snapshot.OperatingIncome = getDuration("Operating Income (Loss)")
	snapshot.InterestExpense = getDuration("Interest Expense")
	snapshot.IncomeTaxExpense = getDuration("Income Tax Expense")
	snapshot.NetIncome = getDuration("Net Income (Loss)")

	// Per Share Metrics (duration)
	snapshot.BasicShares = getDuration("Shares Outstanding (Basic)")
	snapshot.DilutedShares = getDuration("Shares Outstanding (Diluted)")
	snapshot.EPSBasic = getDuration("EPS Basic")
	snapshot.EPSDiluted = getDuration("EPS Diluted")

	// Cash Flow Statement (duration)
	snapshot.CashFlowOperations = getDuration("Cash Flow from Operations")
	snapshot.CashFlowInvesting = getDuration("Cash Flow from Investing")
	snapshot.CashFlowFinancing = getDuration("Cash Flow from Financing")
	snapshot.CapitalExpenditures = getDuration("Capital Expenditures")

	// Non-Cash Items (duration)
	snapshot.DepreciationAmortization = getDuration("Depreciation and Amortization")
	snapshot.StockBasedCompensation = getDuration("Stock-Based Compensation")

	// Validate required fields
	snapshot.MissingRequiredFields = validateRequiredFields(snapshot)

	return snapshot, nil
}

// validateRequiredFields checks if required GAAP fields are present
// Returns a list of missing required field names
func validateRequiredFields(snapshot *FinancialSnapshot) []string {
	var missing []string

	// Map of required field labels to their snapshot values
	requiredFields := map[string]float64{
		"Total Assets":                 snapshot.TotalAssets,
		"Total Liabilities":            snapshot.TotalLiabilities,
		"Stockholders Equity":          snapshot.StockholdersEquity,
		"Revenue":                      snapshot.Revenue,
		"Net Income (Loss)":            snapshot.NetIncome,
		"Cash Flow from Operations":    snapshot.CashFlowOperations,
		"Shares Outstanding (Diluted)": snapshot.DilutedShares,
	}

	// Check each required field
	for label, value := range requiredFields {
		// Zero value indicates the field is missing (or legitimately zero, but that's rare for required fields)
		if value == 0 {
			missing = append(missing, label)
		}
	}

	// Sort for consistent output
	sort.Strings(missing)

	return missing
}

// GetNetIncome returns net income (loss) for the most recent period
func (x *XBRL) GetNetIncome(period string) (float64, error) {
	query := x.Query().
		ByLabel("Net Income (Loss)").
		DurationOnly()

	if period != "" {
		query = query.ForPeriodEndingOn(period)
	}

	fact, err := query.MostRecent()
	if err != nil {
		return 0, nil // Many companies report losses, return 0
	}

	return fact.Float64()
}

// extractMetadata extracts company and document metadata from DEI facts
func extractMetadata(x *XBRL, snapshot *FinancialSnapshot) {
	for _, fact := range x.Facts {
		// Extract company name
		if fact.Concept == "dei:EntityRegistrantName" {
			snapshot.CompanyName = fact.Value
		}

		// Extract CIK
		if fact.Concept == "dei:EntityCentralIndexKey" {
			snapshot.CIK = fact.Value
		}

		// Extract fiscal period (FY for 10-K, Q1/Q2/Q3/Q4 for 10-Q)
		if fact.Concept == "dei:DocumentFiscalPeriodFocus" {
			snapshot.FiscalPeriod = fact.Value
		}

		// Extract document type (10-K, 10-Q, etc.)
		if fact.Concept == "dei:DocumentType" {
			snapshot.FormType = fact.Value
		}
	}
}

// findFiscalYearEnd finds the fiscal year end date from the XBRL contexts
// This is the reporting period end date, not the filing date
func findFiscalYearEnd(x *XBRL) time.Time {
	var latestEnd time.Time

	// Look for the longest duration period (annual for 10-K, quarterly for 10-Q)
	// that has actual financial data (not just a filing date)
	for _, ctx := range x.Contexts {
		// Skip if no period
		if ctx.Period.EndDate == "" && ctx.Period.Instant == "" {
			continue
		}

		// Parse end date
		var endDate time.Time
		var err error

		if ctx.Period.EndDate != "" {
			endDate, err = time.Parse("2006-01-02", ctx.Period.EndDate)
		} else if ctx.Period.Instant != "" {
			endDate, err = time.Parse("2006-01-02", ctx.Period.Instant)
		}

		if err != nil {
			continue
		}

		// For annual reports, look for ~365 day periods
		// For quarterly, look for ~90 day periods
		if ctx.Period.StartDate != "" {
			startDate, err := time.Parse("2006-01-02", ctx.Period.StartDate)
			if err == nil {
				duration := endDate.Sub(startDate).Hours() / 24

				// Annual period (300-400 days)
				if duration >= 300 && duration <= 400 {
					if endDate.After(latestEnd) {
						latestEnd = endDate
					}
				}

				// Quarterly period (80-100 days)
				if duration >= 80 && duration <= 100 {
					if endDate.After(latestEnd) {
						latestEnd = endDate
					}
				}
			}
		}
	}

	return latestEnd
}
