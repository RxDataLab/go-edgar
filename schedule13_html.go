package edgar

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ParseSchedule13HTML parses HTML/XHTML rendered Schedule 13D or 13G filings.
// This handles the modern SEC filing format where data is in HTML tables.
func ParseSchedule13HTML(data []byte) (*Schedule13Filing, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	filing := &Schedule13Filing{}

	// Determine form type (13D vs 13G) from page content
	pageText := extractText(doc)
	if strings.Contains(pageText, "SCHEDULE 13D") {
		filing.FormType = "SC 13D"
	} else if strings.Contains(pageText, "SCHEDULE 13G") {
		filing.FormType = "SC 13G"
	}

	// Check for amendment
	if strings.Contains(pageText, "Amendment No.") || strings.Contains(pageText, "(Amendment No.") {
		filing.IsAmendment = true
		filing.FormType += "/A"
	}

	// Extract issuer information using pattern matching (works with any HTML structure)
	// Try blue divs first (modern XHTML format)
	blueDivs := findBlueInformationDivs(doc)
	if len(blueDivs) >= 4 {
		filing.IssuerName = blueDivs[0]
		filing.SecurityTitle = blueDivs[1]
		filing.IssuerCUSIP = blueDivs[2]
	} else {
		// Fall back to bold tag extraction (old HTML format)
		filing.IssuerName = extractFieldBeforeMarker(pageText, "(Name of Issuer)")
		filing.SecurityTitle = extractFieldBeforeMarker(pageText, "(Title of Class of Securities)")
		filing.IssuerCUSIP = extractFieldBeforeMarker(pageText, "(CUSIP Number)")
	}

	// Extract event date
	eventDate := extractBetween(pageText, "(Date of Event Which Requires Filing of this Statement)", "Check the appropriate box")
	if eventDate == "" {
		eventDate = extractBetween(pageText, "(Date of Event Which Requires Filing of This Statement)", "Check the appropriate box")
	}
	filing.EventDate = strings.TrimSpace(eventDate)

	// Extract reporting persons from HTML tables
	filing.ReportingPersons = extractReportingPersonsHTML(doc)

	// Extract rule designations for 13G
	if strings.Contains(filing.FormType, "13G") {
		if strings.Contains(pageText, "Rule 13d-1(b)") {
			filing.RuleDesignations = append(filing.RuleDesignations, "Rule 13d-1(b)")
		}
		if strings.Contains(pageText, "Rule 13d-1(c)") {
			filing.RuleDesignations = append(filing.RuleDesignations, "Rule 13d-1(c)")
		}
		if strings.Contains(pageText, "Rule 13d-1(d)") {
			filing.RuleDesignations = append(filing.RuleDesignations, "Rule 13d-1(d)")
		}
	}

	return filing, nil
}

// extractReportingPersonsHTML extracts reporting person data from HTML tables.
func extractReportingPersonsHTML(doc *html.Node) []ReportingPerson13 {
	// Try modern XHTML format first (with id="reportingPersonDetails")
	modernTables := findAllTables(doc, "reportingPersonDetails")
	if len(modernTables) > 0 {
		return extractModernXHTMLPersons(modernTables)
	}

	// Fall back to old HTML format (tables with "NAMES OF REPORTING PERSONS")
	return extractOldHTMLPersons(doc)
}

// extractModernXHTMLPersons handles modern XHTML format with styled divs
func extractModernXHTMLPersons(tables []*html.Node) []ReportingPerson13 {
	var persons []ReportingPerson13

	for _, table := range tables {
		person := ReportingPerson13{}

		// Extract all text divs from this specific table
		tableDivs := findAllTextDivsInNode(table)

		// The structure is predictable - iterate through rows
		for i, div := range tableDivs {
			text := strings.TrimSpace(div)

			// First text div is the name
			if i == 0 && person.Name == "" {
				person.Name = text
			}

			// Look for numeric values and assign based on position
			if val := parseInt64(text); val > 0 {
				// Assign based on which numeric field we haven't filled yet
				if person.SoleVotingPower == 0 {
					person.SoleVotingPower = val
				} else if person.SharedVotingPower == 0 && val != person.SoleVotingPower {
					person.SharedVotingPower = val
				} else if person.SoleDispositivePower == 0 && val != person.SharedVotingPower {
					person.SoleDispositivePower = val
				} else if person.SharedDispositivePower == 0 && val != person.SoleDispositivePower {
					person.SharedDispositivePower = val
				} else if person.AggregateAmountOwned == 0 && val != person.SharedDispositivePower {
					person.AggregateAmountOwned = val
				}
			}

			// Look for percentage
			if strings.Contains(text, "%") && person.PercentOfClass == 0.0 {
				percentStr := strings.ReplaceAll(text, "%", "")
				person.PercentOfClass = parseFloat64(percentStr)
			}

			// Look for type codes (IA, PN, HC, OO, etc.)
			if len(text) <= 10 && strings.Contains(text, ",") {
				// Likely a type code like "IA, PN"
				person.TypeOfReportingPerson = text
			}

			// Look for citizenship/state codes
			if len(text) < 30 && (strings.Contains(strings.ToUpper(text), "DELAWARE") ||
				strings.Contains(strings.ToUpper(text), "UNITED STATES") ||
				len(text) == 2) && person.Citizenship == "" {
				person.Citizenship = text
			}
		}

		// Only add if we got meaningful data
		if person.Name != "" && len(person.Name) > 3 {
			persons = append(persons, person)
		}
	}

	return persons
}

// extractOldHTMLPersons handles old HTML format with multiple tables per person
func extractOldHTMLPersons(doc *html.Node) []ReportingPerson13 {
	var persons []ReportingPerson13

	// Get all tables in the document
	allTables := findAllTablesInOrder(doc)

	// Find indices of tables containing "NAMES OF REPORTING PERSONS"
	nameTableIndices := []int{}
	for i, table := range allTables {
		text := extractText(table)
		if strings.Contains(text, "NAMES OF REPORTING PERSONS") {
			nameTableIndices = append(nameTableIndices, i)
		}
	}

	// For each "NAMES" table, combine data from it and the next 2 tables
	for _, idx := range nameTableIndices {
		person := ReportingPerson13{}

		// Table 1: Name and citizenship
		if idx < len(allTables) {
			nameText := extractText(allTables[idx])

			// Extract name - clean up row numbers and extra text
			if name := extractBetween(nameText, "NAMES OF REPORTING PERSONS", "CHECK THE APPROPRIATE BOX"); name != "" {
				name = strings.TrimSpace(name)
				// Remove trailing numbers (row numbers like "2")
				name = regexp.MustCompile(`\s+\d+\s*$`).ReplaceAllString(name, "")
				person.Name = strings.TrimSpace(name)
			}

			// Extract citizenship - simpler, more direct extraction
			if citizenship := extractBetween(nameText, "CITIZENSHIP OR PLACE OF ORGANIZATION", ""); citizenship != "" {
				citizenship = strings.TrimSpace(citizenship)
				// Remove any trailing numbers or excess whitespace
				citizenship = regexp.MustCompile(`\s+\d+\s*$`).ReplaceAllString(citizenship, "")
				citizenship = strings.TrimSpace(citizenship)
				if len(citizenship) > 0 && len(citizenship) < 50 {
					person.Citizenship = citizenship
				}
			}
		}

		// Table 2: Voting and dispositive powers (next table)
		if idx+1 < len(allTables) {
			powersText := extractText(allTables[idx+1])

			// Sole voting power
			if sole := extractBetween(powersText, "SOLE VOTING POWER", "SHARED VOTING POWER"); sole != "" {
				person.SoleVotingPower = parseInt64(sole)
			}

			// Shared voting power
			if shared := extractBetween(powersText, "SHARED VOTING POWER", "SOLE DISPOSITIVE POWER"); shared != "" {
				person.SharedVotingPower = parseInt64(shared)
			}

			// Sole dispositive power
			if soleDisp := extractBetween(powersText, "SOLE DISPOSITIVE POWER", "SHARED DISPOSITIVE POWER"); soleDisp != "" {
				person.SoleDispositivePower = parseInt64(soleDisp)
			}

			// Shared dispositive power
			if sharedDisp := extractBetween(powersText, "SHARED DISPOSITIVE POWER", ""); sharedDisp != "" {
				person.SharedDispositivePower = parseInt64(sharedDisp)
			}
		}

		// Table 3: Aggregate amount, percent, type (two tables after)
		if idx+2 < len(allTables) {
			aggText := extractText(allTables[idx+2])

			// Aggregate amount owned
			if agg := extractBetween(aggText, "AGGREGATE AMOUNT BENEFICIALLY OWNED", "CHECK BOX IF"); agg != "" {
				person.AggregateAmountOwned = parseInt64(agg)
			}

			// Percent of class - extract the number with decimal point and % sign
			if pct := extractBetween(aggText, "PERCENT OF CLASS", "TYPE OF REPORTING PERSON"); pct != "" {
				// Look for pattern like "5.1%" or "12.34%"
				re := regexp.MustCompile(`\d+\.?\d*%`)
				match := re.FindString(pct)
				if match != "" {
					// Remove the % sign and parse
					match = strings.TrimSuffix(match, "%")
					person.PercentOfClass = parseFloat64(match)
				}
			}

			// Type of reporting person - extract after the label, skip "(See Instructions)"
			if typeStr := extractAfterMarker(aggText, "TYPE OF REPORTING PERSON"); typeStr != "" {
				// Skip "(See Instructions)" text and look for actual type codes
				typeStr = strings.ReplaceAll(typeStr, "(See Instructions)", "")
				lines := strings.Split(typeStr, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					// Look for lines that look like type codes (letters and commas, not too long)
					if line != "" && len(line) < 30 && !strings.Contains(line, "Page") && !strings.Contains(line, "CUSIP") {
						person.TypeOfReportingPerson = line
						break
					}
				}
			}
		}

		// Only add if we got meaningful data
		if person.Name != "" && len(person.Name) > 3 {
			persons = append(persons, person)
		}
	}

	return persons
}

// findAllTextDivs finds all <div class="text"> elements
func findAllTextDivs(n *html.Node) []string {
	var texts []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && attr.Val == "text" {
					text := extractText(n)
					texts = append(texts, strings.TrimSpace(text))
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return texts
}

// findAllTextDivsInNode finds text divs within a specific node
func findAllTextDivsInNode(n *html.Node) []string {
	return findAllTextDivs(n)
}

// findBlueInformationDivs finds blue information divs on the cover page
func findBlueInformationDivs(n *html.Node) []string {
	var divs []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && attr.Val == "information" {
					// Check if it has color:blue style
					for _, a2 := range n.Attr {
						if a2.Key == "style" && strings.Contains(a2.Val, "color:blue") {
							text := extractText(n)
							divs = append(divs, strings.TrimSpace(text))
							break
						}
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return divs
}

// extractTextValue extracts text from <div class="text"> or similar
func extractTextValue(s string) string {
	// Look for text within div tags
	re := regexp.MustCompile(`<div[^>]*class="text"[^>]*>([^<]+)</div>`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Fall back to stripping all HTML tags
	re = regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(s, " ")
	return strings.TrimSpace(text)
}

// findAllTables finds all tables with a specific id pattern
func findAllTables(n *html.Node, idPattern string) []*html.Node {
	var tables []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			for _, attr := range n.Attr {
				if attr.Key == "id" && strings.Contains(attr.Val, idPattern) {
					tables = append(tables, n)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return tables
}

// findTablesContaining finds all tables that contain specific text
func findTablesContaining(n *html.Node, text string) []*html.Node {
	var tables []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			tableText := extractText(n)
			if strings.Contains(tableText, text) {
				tables = append(tables, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return tables
}

// findAllTablesInOrder finds all tables in document order
func findAllTablesInOrder(n *html.Node) []*html.Node {
	var tables []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			tables = append(tables, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return tables
}

// extractTableCells extracts all text content from table cells
func extractTableCells(table *html.Node) []string {
	var cells []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "td" || n.Data == "th") {
			// Get the full HTML of this cell
			var buf strings.Builder
			renderNode(&buf, n)
			cells = append(cells, buf.String())
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(table)
	return cells
}

// renderNode renders an HTML node to string
func renderNode(w io.Writer, n *html.Node) {
	html.Render(w, n)
}

// extractText extracts all text content from HTML
func extractText(n *html.Node) string {
	var buf strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
			buf.WriteString(" ")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return buf.String()
}

// extractBetween extracts text between two markers
func extractBetween(text, start, end string) string {
	startIdx := strings.Index(text, start)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)

	var result string
	if end == "" {
		// Extract to end of string (with reasonable limit)
		result = text[startIdx:]
		if len(result) > 200 {
			result = result[:200]
		}
	} else {
		endIdx := strings.Index(text[startIdx:], end)
		if endIdx == -1 {
			return ""
		}
		result = text[startIdx : startIdx+endIdx]
	}

	result = strings.TrimSpace(result)

	// Clean up whitespace
	re := regexp.MustCompile(`\s+`)
	result = re.ReplaceAllString(result, " ")

	return result
}

// extractFieldBeforeMarker finds the value before a marker like "(Name of Issuer)"
func extractFieldBeforeMarker(text, marker string) string {
	idx := strings.Index(text, marker)
	if idx == -1 {
		return ""
	}

	// Look backwards for the value (usually within 200 chars)
	start := idx - 200
	if start < 0 {
		start = 0
	}

	chunk := text[start:idx]
	lines := strings.Split(chunk, "\n")

	// Look for non-empty lines going backwards
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.HasPrefix(line, "(") && len(line) > 2 {
			return line
		}
	}

	return ""
}

// extractAfterMarker finds the value after a marker
func extractAfterMarker(text, marker string) string {
	idx := strings.Index(text, marker)
	if idx == -1 {
		return ""
	}

	// Start after the marker
	start := idx + len(marker)
	if start >= len(text) {
		return ""
	}

	// Look for the next 200 chars
	end := start + 200
	if end > len(text) {
		end = len(text)
	}

	chunk := text[start:end]
	return strings.TrimSpace(chunk)
}

// ParseSchedule13Auto automatically detects format (XML vs HTML) and parses
func ParseSchedule13Auto(data []byte) (*Schedule13Filing, error) {
	// Try pure XML first
	dataStr := string(data)

	// Check if it's pure XML (starts with <?xml and has edgarSubmission root)
	if strings.HasPrefix(strings.TrimSpace(dataStr), "<?xml") &&
		strings.Contains(dataStr, "<edgarSubmission") &&
		!strings.Contains(dataStr, "<!DOCTYPE html") {

		// Determine 13D vs 13G by namespace
		if strings.Contains(dataStr, "schedule13D") {
			return ParseSchedule13D(data)
		} else if strings.Contains(dataStr, "schedule13g") {
			return ParseSchedule13G(data)
		}
	}

	// Otherwise, parse as HTML/XHTML
	return ParseSchedule13HTML(data)
}
