package edgar

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Schedule13Filing represents a parsed SC 13D or SC 13G filing.
// These forms report beneficial ownership of 5% or more of a company's stock.
//
// SC 13D = Active/Activist investor (detailed narrative required)
// SC 13G = Passive institutional investor (simpler reporting)
type Schedule13Filing struct {
	// Form metadata
	FormType        string // "SC 13D", "SC 13D/A", "SC 13G", "SC 13G/A"
	IsAmendment     bool   // true if contains "/A"
	AmendmentNumber *int   // nil for original, 1, 2, 3... for numbered amendments
	FilingDate      string // From filing metadata (not in XML)

	// Issuer (company being reported on)
	IssuerCIK   string
	IssuerName  string
	IssuerCUSIP string

	// Security information
	SecurityTitle string

	// Reporting persons (investors filing the report)
	ReportingPersons []ReportingPerson13

	// Narrative content (polymorphic - either 13D or 13G items)
	Items13D *Schedule13DItems // nil if this is a 13G
	Items13G *Schedule13GItems // nil if this is a 13D

	// 13D specific fields
	DateOfEvent     string // Event triggering filing (13D only)
	PreviouslyFiled bool   // Indicates amendment (13D only)

	// 13G specific fields
	EventDate        string   // Event date requiring filing (13G only)
	RuleDesignations []string // Rule 13d-1(b), (c), etc. (13G only)

	// Filer CIK from header (fallback when reportingPersonCIK is missing)
	FilerCIK string
}

// ReportingPerson13 represents an individual or entity reporting beneficial ownership.
type ReportingPerson13 struct {
	CIK   string // May be empty for foreign entities or when only in header
	Name  string
	NoCIK bool // true for foreign entities without CIK

	// Ownership amounts
	AggregateAmountOwned int64   // Total shares owned
	PercentOfClass       float64 // Ownership percentage

	// Voting power
	SoleVotingPower   int64 // Shares with sole voting control
	SharedVotingPower int64 // Shares with shared voting control

	// Dispositive power (ability to dispose/sell)
	SoleDispositivePower   int64 // Shares with sole disposition control
	SharedDispositivePower int64 // Shares with shared disposition control

	// Critical for aggregation logic
	MemberOfGroup      string // "a" = joint filer (don't sum), "b" = separate filer
	IsAggregateExclude bool   // Exclude from total count

	// Metadata
	TypeOfReportingPerson string // "IN" (individual), "CO" (corp), "PN" (partnership), etc.
	FundType              string // "WC" (working capital), "PF" (pension fund), etc.
	Citizenship           string
	Comment               string // Relationship explanations
}

// Schedule13DItems contains Items 1-7 from Schedule 13D.
// Item 4 (Purpose of Transaction) is the most important for activist analysis.
type Schedule13DItems struct {
	// Item 1: Security and Issuer
	Item1SecurityTitle string
	Item1IssuerName    string
	Item1IssuerAddress string

	// Item 2: Identity and Background
	Item2FilingPersons       string
	Item2BusinessAddress     string
	Item2PrincipalOccupation string
	Item2Convictions         string
	Item2Citizenship         string

	// Item 3: Source and Amount of Funds
	Item3SourceOfFunds string

	// Item 4: Purpose of Transaction (MOST IMPORTANT)
	// Contains activist intent, board letters, future plans, etc.
	Item4PurposeOfTransaction string

	// Item 5: Interest in Securities of the Issuer
	Item5PercentageOfClass string
	Item5NumberOfShares    string
	Item5Transactions      string
	Item5Shareholders      string
	Item5Date5PctOwnership string

	// Item 6: Contracts, Arrangements, Understandings
	Item6Contracts string

	// Item 7: Material to be Filed as Exhibits
	Item7Exhibits string
}

// Schedule13GItems contains Items 1-10 from Schedule 13G.
// Item 10 (Certification) is key - certifies passive investor status.
type Schedule13GItems struct {
	// Item 1: Name and address of issuer
	Item1IssuerName    string
	Item1IssuerAddress string

	// Item 2: Name and address of person filing
	Item2FilerNames     string
	Item2FilerAddresses string
	Item2Citizenship    string

	// Item 3: If applicable (usually N/A)
	Item3NotApplicable bool

	// Item 4: Ownership
	Item4AmountBeneficiallyOwned string
	Item4PercentOfClass          string
	Item4SoleVoting              string
	Item4SharedVoting            string
	Item4SoleDispositive         string
	Item4SharedDispositive       string

	// Item 5: Ownership of 5% or less
	Item5NotApplicable       bool
	Item5Ownership5PctOrLess string

	// Item 6: Ownership of more than 5%
	Item6NotApplicable bool

	// Item 7: Identification and classification
	Item7NotApplicable bool

	// Item 8: Identification and classification of members
	Item8NotApplicable bool

	// Item 9: Notice pursuant to Rule 13d-1(k)
	Item9NotApplicable bool

	// Item 10: Certification (important - passive investor cert)
	Item10Certification string
}

// TotalVotingPower returns total voting power (sole + shared).
func (r *ReportingPerson13) TotalVotingPower() int64 {
	return r.SoleVotingPower + r.SharedVotingPower
}

// TotalDispositivePower returns total dispositive power (sole + shared).
func (r *ReportingPerson13) TotalDispositivePower() int64 {
	return r.SoleDispositivePower + r.SharedDispositivePower
}

// CalculateTotalShares correctly aggregates shares across all reporting persons.
// Critical: handles joint filers (memberOfGroup="a") to avoid double-counting.
func (s *Schedule13Filing) CalculateTotalShares() int64 {
	// Step 1: Exclude flagged shares
	included := []ReportingPerson13{}
	for _, p := range s.ReportingPersons {
		if !p.IsAggregateExclude {
			included = append(included, p)
		}
	}

	// Step 2: Check for joint filers
	groupMembers := []ReportingPerson13{}
	for _, p := range included {
		if p.MemberOfGroup == "a" {
			groupMembers = append(groupMembers, p)
		}
	}

	if len(groupMembers) > 0 {
		// Joint filers: all report same shares - take max (not sum!)
		max := int64(0)
		for _, p := range groupMembers {
			if p.AggregateAmountOwned > max {
				max = p.AggregateAmountOwned
			}
		}
		return max
	}

	// Separate filers: sum all positions
	total := int64(0)
	for _, p := range included {
		total += p.AggregateAmountOwned
	}
	return total
}

// CalculateTotalPercent returns the maximum ownership percentage.
func (s *Schedule13Filing) CalculateTotalPercent() float64 {
	max := 0.0
	for _, p := range s.ReportingPersons {
		if p.PercentOfClass > max {
			max = p.PercentOfClass
		}
	}
	return max
}

// IsActivist returns true if this is a Schedule 13D (active/activist investor).
func (s *Schedule13Filing) IsActivist() bool {
	return strings.Contains(s.FormType, "13D")
}

// IsPassive returns true if this is a Schedule 13G (passive investor).
func (s *Schedule13Filing) IsPassive() bool {
	return strings.Contains(s.FormType, "13G")
}

// ExtractAmendmentInfo parses the form type to determine if it's an amendment
// and extracts the amendment number if present.
func ExtractAmendmentInfo(formType string) (isAmendment bool, amendmentNumber *int) {
	isAmendment = strings.Contains(formType, "/A")
	if !isAmendment {
		return false, nil
	}

	// Try to extract number from patterns like:
	// "SC 13D/A", "SC 13D/A 2", "SCHEDULE 13D/A", "Amendment No. 9", etc.

	// Pattern 1: "Amendment No. 9"
	re := regexp.MustCompile(`Amendment\s+No\.\s+(\d+)`)
	if matches := re.FindStringSubmatch(formType); matches != nil {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			return true, &num
		}
	}

	// Pattern 2: "/A 9" or "/A#9"
	re = regexp.MustCompile(`/A\s*#?(\d+)`)
	if matches := re.FindStringSubmatch(formType); matches != nil {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			return true, &num
		}
	}

	// Amendment without number - return nil
	return true, nil
}

// XML parsing structures for Schedule 13D
// xmlns="http://www.sec.gov/edgar/schedule13D"

type schedule13DXML struct {
	XMLName    xml.Name            `xml:"edgarSubmission"`
	HeaderData schedule13DHeader   `xml:"headerData"`
	FormData   schedule13DFormData `xml:"formData"`
}

type schedule13DHeader struct {
	SubmissionType string           `xml:"submissionType"`
	FilerInfo      schedule13DFiler `xml:"filerInfo"`
}

type schedule13DFiler struct {
	Filer struct {
		FilerCredentials struct {
			CIK string `xml:"cik"`
		} `xml:"filerCredentials"`
	} `xml:"filer"`
}

type schedule13DFormData struct {
	CoverPageHeader  schedule13DCover            `xml:"coverPageHeader"`
	ReportingPersons schedule13DReportingPersons `xml:"reportingPersons"`
	Items1To7        schedule13DItems1To7        `xml:"items1To7"`
}

type schedule13DCover struct {
	SecuritiesClassTitle string `xml:"securitiesClassTitle"`
	DateOfEvent          string `xml:"dateOfEvent"`
	PreviouslyFiledFlag  string `xml:"previouslyFiledFlag"`
	IssuerInfo           struct {
		IssuerCIK   string `xml:"issuerCIK"`
		IssuerCUSIP string `xml:"issuerCUSIP"`
		IssuerName  string `xml:"issuerName"`
	} `xml:"issuerInfo"`
}

type schedule13DReportingPersons struct {
	ReportingPersonInfo []schedule13DReportingPerson `xml:"reportingPersonInfo"`
}

type schedule13DReportingPerson struct {
	ReportingPersonCIK        string `xml:"reportingPersonCIK"`
	ReportingPersonName       string `xml:"reportingPersonName"`
	ReportingPersonNoCIK      string `xml:"reportingPersonNoCIK"`
	FundType                  string `xml:"fundType"`
	CitizenshipOrOrganization string `xml:"citizenshipOrOrganization"`
	SoleVotingPower           string `xml:"soleVotingPower"`
	SharedVotingPower         string `xml:"sharedVotingPower"`
	SoleDispositivePower      string `xml:"soleDispositivePower"`
	SharedDispositivePower    string `xml:"sharedDispositivePower"`
	AggregateAmountOwned      string `xml:"aggregateAmountOwned"`
	IsAggregateExcludeShares  string `xml:"isAggregateExcludeShares"`
	PercentOfClass            string `xml:"percentOfClass"`
	TypeOfReportingPerson     string `xml:"typeOfReportingPerson"`
	MemberOfGroup             string `xml:"memberOfGroup"`
	CommentContent            string `xml:"commentContent"`
}

type schedule13DItems1To7 struct {
	Item1 struct {
		SecurityTitle          string `xml:"securityTitle"`
		IssuerName             string `xml:"issuerName"`
		IssuerPrincipalAddress string `xml:"issuerPrincipalAddress"`
	} `xml:"item1"`
	Item2 struct {
		FilingPersonName         string `xml:"filingPersonName"`
		PrincipalBusinessAddress string `xml:"principalBusinessAddress"`
		PrincipalJob             string `xml:"principalJob"`
		HasBeenConvicted         string `xml:"hasBeenConvicted"`
		Citizenship              string `xml:"citizenship"`
	} `xml:"item2"`
	Item3 struct {
		FundsSource string `xml:"fundsSource"`
	} `xml:"item3"`
	Item4 struct {
		TransactionPurpose string `xml:"transactionPurpose"`
	} `xml:"item4"`
	Item5 struct {
		PercentageOfClassSecurities string `xml:"percentageOfClassSecurities"`
		NumberOfShares              string `xml:"numberOfShares"`
		TransactionDesc             string `xml:"transactionDesc"`
		ListOfShareholders          string `xml:"listOfShareholders"`
		Date5PercentOwnership       string `xml:"date5PercentOwnership"`
	} `xml:"item5"`
	Item6 struct {
		ContractDescription string `xml:"contractDescription"`
	} `xml:"item6"`
	Item7 struct {
		FiledExhibits string `xml:"filedExhibits"`
	} `xml:"item7"`
}

// ParseSchedule13D parses a Schedule 13D XML filing.
func ParseSchedule13D(data []byte) (*Schedule13Filing, error) {
	var xmlDoc schedule13DXML
	if err := xml.Unmarshal(data, &xmlDoc); err != nil {
		return nil, fmt.Errorf("failed to parse Schedule 13D XML: %w", err)
	}

	filing := &Schedule13Filing{
		FormType:        xmlDoc.HeaderData.SubmissionType,
		FilerCIK:        xmlDoc.HeaderData.FilerInfo.Filer.FilerCredentials.CIK,
		IssuerCIK:       xmlDoc.FormData.CoverPageHeader.IssuerInfo.IssuerCIK,
		IssuerName:      xmlDoc.FormData.CoverPageHeader.IssuerInfo.IssuerName,
		IssuerCUSIP:     xmlDoc.FormData.CoverPageHeader.IssuerInfo.IssuerCUSIP,
		SecurityTitle:   xmlDoc.FormData.CoverPageHeader.SecuritiesClassTitle,
		DateOfEvent:     xmlDoc.FormData.CoverPageHeader.DateOfEvent,
		PreviouslyFiled: strings.ToUpper(xmlDoc.FormData.CoverPageHeader.PreviouslyFiledFlag) == "TRUE",
	}

	// Extract amendment info
	filing.IsAmendment, filing.AmendmentNumber = ExtractAmendmentInfo(filing.FormType)

	// Parse reporting persons
	for _, personXML := range xmlDoc.FormData.ReportingPersons.ReportingPersonInfo {
		person := ReportingPerson13{
			CIK:                   personXML.ReportingPersonCIK,
			Name:                  personXML.ReportingPersonName,
			NoCIK:                 strings.ToUpper(personXML.ReportingPersonNoCIK) == "Y",
			FundType:              personXML.FundType,
			Citizenship:           personXML.CitizenshipOrOrganization,
			TypeOfReportingPerson: personXML.TypeOfReportingPerson,
			MemberOfGroup:         personXML.MemberOfGroup,
			IsAggregateExclude:    strings.ToUpper(personXML.IsAggregateExcludeShares) == "Y",
			Comment:               personXML.CommentContent,
		}

		// Parse numeric fields
		person.SoleVotingPower = parseInt64(personXML.SoleVotingPower)
		person.SharedVotingPower = parseInt64(personXML.SharedVotingPower)
		person.SoleDispositivePower = parseInt64(personXML.SoleDispositivePower)
		person.SharedDispositivePower = parseInt64(personXML.SharedDispositivePower)
		person.AggregateAmountOwned = parseInt64(personXML.AggregateAmountOwned)
		person.PercentOfClass = parseFloat64(personXML.PercentOfClass)

		// Fallback to filer CIK if reporting person CIK is empty
		if person.CIK == "" && !person.NoCIK {
			person.CIK = filing.FilerCIK
		}

		filing.ReportingPersons = append(filing.ReportingPersons, person)
	}

	// Parse Items 1-7
	items := &Schedule13DItems{
		Item1SecurityTitle:        xmlDoc.FormData.Items1To7.Item1.SecurityTitle,
		Item1IssuerName:           xmlDoc.FormData.Items1To7.Item1.IssuerName,
		Item1IssuerAddress:        xmlDoc.FormData.Items1To7.Item1.IssuerPrincipalAddress,
		Item2FilingPersons:        xmlDoc.FormData.Items1To7.Item2.FilingPersonName,
		Item2BusinessAddress:      xmlDoc.FormData.Items1To7.Item2.PrincipalBusinessAddress,
		Item2PrincipalOccupation:  xmlDoc.FormData.Items1To7.Item2.PrincipalJob,
		Item2Convictions:          xmlDoc.FormData.Items1To7.Item2.HasBeenConvicted,
		Item2Citizenship:          xmlDoc.FormData.Items1To7.Item2.Citizenship,
		Item3SourceOfFunds:        xmlDoc.FormData.Items1To7.Item3.FundsSource,
		Item4PurposeOfTransaction: xmlDoc.FormData.Items1To7.Item4.TransactionPurpose,
		Item5PercentageOfClass:    xmlDoc.FormData.Items1To7.Item5.PercentageOfClassSecurities,
		Item5NumberOfShares:       xmlDoc.FormData.Items1To7.Item5.NumberOfShares,
		Item5Transactions:         xmlDoc.FormData.Items1To7.Item5.TransactionDesc,
		Item5Shareholders:         xmlDoc.FormData.Items1To7.Item5.ListOfShareholders,
		Item5Date5PctOwnership:    xmlDoc.FormData.Items1To7.Item5.Date5PercentOwnership,
		Item6Contracts:            xmlDoc.FormData.Items1To7.Item6.ContractDescription,
		Item7Exhibits:             xmlDoc.FormData.Items1To7.Item7.FiledExhibits,
	}
	filing.Items13D = items

	return filing, nil
}

// Helper functions for parsing numeric values

func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Handle "-0-" as zero
	if strings.Contains(s, "-0-") {
		return 0
	}

	// Extract first number from string (handles cases like "1,874,978 6" or "text 123,456 more text")
	re := regexp.MustCompile(`[0-9,]+`)
	match := re.FindString(s)
	if match == "" {
		return 0
	}

	// Remove commas
	match = strings.ReplaceAll(match, ",", "")

	// Parse as int
	if val, err := strconv.ParseInt(match, 10, 64); err == nil {
		return val
	}

	// Fallback: try parsing as float and convert
	if f, err := strconv.ParseFloat(match, 64); err == nil {
		return int64(f)
	}

	return 0
}

func parseFloat64(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0.0
	}

	// Handle "-0-" as zero
	if strings.Contains(s, "-0-") {
		return 0.0
	}

	// Extract first number from string (handles "5.1% (1)" or "text 12.34 more text")
	re := regexp.MustCompile(`[0-9,]+\.?[0-9]*`)
	match := re.FindString(s)
	if match == "" {
		return 0.0
	}

	// Remove commas
	match = strings.ReplaceAll(match, ",", "")

	if f, err := strconv.ParseFloat(match, 64); err == nil {
		return f
	}

	return 0.0
}

// XML parsing structures for Schedule 13G
// xmlns="http://www.sec.gov/edgar/schedule13g"
// Note: 13G has different element names than 13D!

type schedule13GXML struct {
	XMLName    xml.Name            `xml:"edgarSubmission"`
	HeaderData schedule13GHeader   `xml:"headerData"`
	FormData   schedule13GFormData `xml:"formData"`
}

type schedule13GHeader struct {
	SubmissionType string           `xml:"submissionType"`
	FilerInfo      schedule13GFiler `xml:"filerInfo"`
}

type schedule13GFiler struct {
	Filer struct {
		FilerCredentials struct {
			CIK string `xml:"cik"`
		} `xml:"filerCredentials"`
	} `xml:"filer"`
}

type schedule13GFormData struct {
	CoverPageHeader                       schedule13GCover             `xml:"coverPageHeader"`
	CoverPageHeaderReportingPersonDetails []schedule13GReportingPerson `xml:"coverPageHeaderReportingPersonDetails"`
	Items                                 schedule13GItems             `xml:"items"`
}

type schedule13GCover struct {
	SecuritiesClassTitle                 string `xml:"securitiesClassTitle"`
	EventDateRequiresFilingThisStatement string `xml:"eventDateRequiresFilingThisStatement"`
	IssuerInfo                           struct {
		IssuerCik   string `xml:"issuerCik"` // Note: different case than 13D!
		IssuerName  string `xml:"issuerName"`
		IssuerCusip string `xml:"issuerCusip"` // Note: different case than 13D!
	} `xml:"issuerInfo"`
	DesignateRulesPursuantThisScheduleFiled struct {
		DesignateRulePursuantThisScheduleFiled []string `xml:"designateRulePursuantThisScheduleFiled"`
	} `xml:"designateRulesPursuantThisScheduleFiled"`
}

type schedule13GReportingPerson struct {
	ReportingPersonName                            string `xml:"reportingPersonName"`
	ReportingPersonNoCIK                           string `xml:"reportingPersonNoCIK"`
	CitizenshipOrOrganization                      string `xml:"citizenshipOrOrganization"`
	ReportingPersonBeneficiallyOwnedNumberOfShares struct {
		SoleVotingPower        string `xml:"soleVotingPower"`
		SharedVotingPower      string `xml:"sharedVotingPower"`
		SoleDispositivePower   string `xml:"soleDispositivePower"`
		SharedDispositivePower string `xml:"sharedDispositivePower"`
	} `xml:"reportingPersonBeneficiallyOwnedNumberOfShares"`
	ReportingPersonBeneficiallyOwnedAggregateNumberOfShares string `xml:"reportingPersonBeneficiallyOwnedAggregateNumberOfShares"`
	ClassPercent                                            string `xml:"classPercent"` // Note: different from 13D!
	MemberGroup                                             string `xml:"memberGroup"`  // Note: different from 13D!
	TypeOfReportingPerson                                   string `xml:"typeOfReportingPerson"`
	IsAggregateExcludeShares                                string `xml:"isAggregateExcludeShares"`
}

type schedule13GItems struct {
	Item1 struct {
		IssuerName                            string `xml:"issuerName"`
		IssuerPrincipalExecutiveOfficeAddress string `xml:"issuerPrincipalExecutiveOfficeAddress"`
	} `xml:"item1"`
	Item2 struct {
		FilingPersonName                          string `xml:"filingPersonName"`
		PrincipalBusinessOfficeOrResidenceAddress string `xml:"principalBusinessOfficeOrResidenceAddress"`
		Citizenship                               string `xml:"citizenship"`
	} `xml:"item2"`
	Item3 struct {
		NotApplicableFlag string `xml:"notApplicableFlag"`
	} `xml:"item3"`
	Item4 struct {
		AmountBeneficiallyOwned string `xml:"amountBeneficiallyOwned"`
		ClassPercent            string `xml:"classPercent"`
		NumberOfSharesPersonHas struct {
			SolePowerOrDirectToVote      string `xml:"solePowerOrDirectToVote"`
			SharedPowerOrDirectToVote    string `xml:"sharedPowerOrDirectToVote"`
			SolePowerOrDirectToDispose   string `xml:"solePowerOrDirectToDispose"`
			SharedPowerOrDirectToDispose string `xml:"sharedPowerOrDirectToDispose"`
		} `xml:"numberOfSharesPersonHas"`
	} `xml:"item4"`
	Item5 struct {
		NotApplicableFlag   string `xml:"notApplicableFlag"`
		Ownership5PctOrLess string `xml:"ownership5PctOrLess"`
	} `xml:"item5"`
	Item6 struct {
		NotApplicableFlag string `xml:"notApplicableFlag"`
	} `xml:"item6"`
	Item7 struct {
		NotApplicableFlag string `xml:"notApplicableFlag"`
	} `xml:"item7"`
	Item8 struct {
		NotApplicableFlag string `xml:"notApplicableFlag"`
	} `xml:"item8"`
	Item9 struct {
		NotApplicableFlag string `xml:"notApplicableFlag"`
	} `xml:"item9"`
	Item10 struct {
		Certifications string `xml:"certifications"`
	} `xml:"item10"`
}

// ParseSchedule13G parses a Schedule 13G XML filing.
func ParseSchedule13G(data []byte) (*Schedule13Filing, error) {
	var xmlDoc schedule13GXML
	if err := xml.Unmarshal(data, &xmlDoc); err != nil {
		return nil, fmt.Errorf("failed to parse Schedule 13G XML: %w", err)
	}

	filing := &Schedule13Filing{
		FormType:         xmlDoc.HeaderData.SubmissionType,
		FilerCIK:         xmlDoc.HeaderData.FilerInfo.Filer.FilerCredentials.CIK,
		IssuerCIK:        xmlDoc.FormData.CoverPageHeader.IssuerInfo.IssuerCik,
		IssuerName:       xmlDoc.FormData.CoverPageHeader.IssuerInfo.IssuerName,
		IssuerCUSIP:      xmlDoc.FormData.CoverPageHeader.IssuerInfo.IssuerCusip,
		SecurityTitle:    xmlDoc.FormData.CoverPageHeader.SecuritiesClassTitle,
		EventDate:        xmlDoc.FormData.CoverPageHeader.EventDateRequiresFilingThisStatement,
		RuleDesignations: xmlDoc.FormData.CoverPageHeader.DesignateRulesPursuantThisScheduleFiled.DesignateRulePursuantThisScheduleFiled,
	}

	// Extract amendment info
	filing.IsAmendment, filing.AmendmentNumber = ExtractAmendmentInfo(filing.FormType)

	// Parse reporting persons
	for _, personXML := range xmlDoc.FormData.CoverPageHeaderReportingPersonDetails {
		person := ReportingPerson13{
			Name:                  personXML.ReportingPersonName,
			NoCIK:                 strings.ToUpper(personXML.ReportingPersonNoCIK) == "Y",
			Citizenship:           personXML.CitizenshipOrOrganization,
			TypeOfReportingPerson: personXML.TypeOfReportingPerson,
			MemberOfGroup:         personXML.MemberGroup, // Note: different element name!
			IsAggregateExclude:    strings.ToUpper(personXML.IsAggregateExcludeShares) == "Y",
		}

		// Parse numeric fields
		person.SoleVotingPower = parseInt64(personXML.ReportingPersonBeneficiallyOwnedNumberOfShares.SoleVotingPower)
		person.SharedVotingPower = parseInt64(personXML.ReportingPersonBeneficiallyOwnedNumberOfShares.SharedVotingPower)
		person.SoleDispositivePower = parseInt64(personXML.ReportingPersonBeneficiallyOwnedNumberOfShares.SoleDispositivePower)
		person.SharedDispositivePower = parseInt64(personXML.ReportingPersonBeneficiallyOwnedNumberOfShares.SharedDispositivePower)
		person.AggregateAmountOwned = parseInt64(personXML.ReportingPersonBeneficiallyOwnedAggregateNumberOfShares)
		person.PercentOfClass = parseFloat64(personXML.ClassPercent)

		// Fallback to filer CIK (13G often doesn't have CIK in person details)
		if person.CIK == "" && !person.NoCIK {
			person.CIK = filing.FilerCIK
		}

		filing.ReportingPersons = append(filing.ReportingPersons, person)
	}

	// Parse Items 1-10
	items := &Schedule13GItems{
		Item1IssuerName:              xmlDoc.FormData.Items.Item1.IssuerName,
		Item1IssuerAddress:           xmlDoc.FormData.Items.Item1.IssuerPrincipalExecutiveOfficeAddress,
		Item2FilerNames:              xmlDoc.FormData.Items.Item2.FilingPersonName,
		Item2FilerAddresses:          xmlDoc.FormData.Items.Item2.PrincipalBusinessOfficeOrResidenceAddress,
		Item2Citizenship:             xmlDoc.FormData.Items.Item2.Citizenship,
		Item3NotApplicable:           strings.ToUpper(xmlDoc.FormData.Items.Item3.NotApplicableFlag) == "Y",
		Item4AmountBeneficiallyOwned: xmlDoc.FormData.Items.Item4.AmountBeneficiallyOwned,
		Item4PercentOfClass:          xmlDoc.FormData.Items.Item4.ClassPercent,
		Item4SoleVoting:              xmlDoc.FormData.Items.Item4.NumberOfSharesPersonHas.SolePowerOrDirectToVote,
		Item4SharedVoting:            xmlDoc.FormData.Items.Item4.NumberOfSharesPersonHas.SharedPowerOrDirectToVote,
		Item4SoleDispositive:         xmlDoc.FormData.Items.Item4.NumberOfSharesPersonHas.SolePowerOrDirectToDispose,
		Item4SharedDispositive:       xmlDoc.FormData.Items.Item4.NumberOfSharesPersonHas.SharedPowerOrDirectToDispose,
		Item5NotApplicable:           strings.ToUpper(xmlDoc.FormData.Items.Item5.NotApplicableFlag) == "Y",
		Item5Ownership5PctOrLess:     xmlDoc.FormData.Items.Item5.Ownership5PctOrLess,
		Item6NotApplicable:           strings.ToUpper(xmlDoc.FormData.Items.Item6.NotApplicableFlag) == "Y",
		Item7NotApplicable:           strings.ToUpper(xmlDoc.FormData.Items.Item7.NotApplicableFlag) == "Y",
		Item8NotApplicable:           strings.ToUpper(xmlDoc.FormData.Items.Item8.NotApplicableFlag) == "Y",
		Item9NotApplicable:           strings.ToUpper(xmlDoc.FormData.Items.Item9.NotApplicableFlag) == "Y",
		Item10Certification:          xmlDoc.FormData.Items.Item10.Certifications,
	}
	filing.Items13G = items

	return filing, nil
}
