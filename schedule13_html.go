package edgar

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
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

	// Extract issuer information from HTML
	// Try multiple extraction strategies in order of reliability:

	// 1. Try extracting from Item 1(a) (most reliable when present)
	filing.IssuerName = extractFromItem1a(doc, pageText)

	// 2. If Item 1(a) not found, extract from cover page <B> tags before markers
	if filing.IssuerName == "" {
		filing.IssuerName = extractBoldBeforeMarker(doc, "(Name of Issuer)")
	}

	// Security title and CUSIP always from cover page
	// Note: &nbsp; in HTML is converted to \u00a0 (non-breaking space) by the parser
	filing.SecurityTitle = extractBoldBeforeMarker(doc, "(Title of Class of Securities)")
	if filing.SecurityTitle == "" {
		// Try with non-breaking space
		filing.SecurityTitle = extractBoldBeforeMarker(doc, "(Title of Class\u00a0of Securities)")
	}

	filing.IssuerCUSIP = extractBoldBeforeMarker(doc, "(CUSIP Number)")
	if filing.IssuerCUSIP == "" {
		// Try with lowercase "number"
		filing.IssuerCUSIP = extractBoldBeforeMarker(doc, "(CUSIP number)")
	}

	// Clean up extracted values
	filing.IssuerName = strings.TrimSpace(filing.IssuerName)
	filing.SecurityTitle = strings.TrimSpace(filing.SecurityTitle)
	filing.IssuerCUSIP = strings.TrimSpace(filing.IssuerCUSIP)

	// Remove footnote markers from CUSIP (e.g., "088786108**" -> "088786108")
	filing.IssuerCUSIP = regexp.MustCompile(`[*†‡§]+$`).ReplaceAllString(filing.IssuerCUSIP, "")

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

	// Extract narrative Items based on form type
	if strings.Contains(filing.FormType, "13D") {
		filing.Items13D = extractSchedule13DItems(doc)
	} else if strings.Contains(filing.FormType, "13G") {
		filing.Items13G = extractSchedule13GItems(doc)
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

		// Clean up person name (remove trailing row numbers like "2.", "3.", etc.)
		if person.Name != "" {
			person.Name = cleanReportingPersonName(person.Name)
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

		// Clean up person name (remove trailing row numbers like "2.", "3.", etc.)
		if person.Name != "" {
			person.Name = cleanReportingPersonName(person.Name)
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

// extractFromItem1a extracts the issuer name from Item 1(a) narrative section
// This is more reliable than cover page extraction when present
func extractFromItem1a(doc *html.Node, pageText string) string {
	// Look for "Item 1(a)" followed by "Name of Issuer:"
	if !strings.Contains(pageText, "Item 1(a)") && !strings.Contains(pageText, "Item 1a") {
		return ""
	}

	// Extract text between "Name of Issuer:" and next "Item" or table marker
	start := strings.Index(pageText, "Name of Issuer:")
	if start == -1 {
		return ""
	}
	start += len("Name of Issuer:")

	// Find end of this section (next Item or significant marker)
	end := start + 500
	if end > len(pageText) {
		end = len(pageText)
	}

	chunk := pageText[start:end]

	// Find the next "Item" to limit our search
	if idx := strings.Index(chunk, "Item 1(b)"); idx != -1 {
		chunk = chunk[:idx]
	} else if idx := strings.Index(chunk, "Item 1b"); idx != -1 {
		chunk = chunk[:idx]
	}

	// Extract first non-empty line that looks like a company name
	lines := strings.Split(chunk, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines, nbsp, and lines that are too short
		if line == "" || line == "&nbsp;" || len(line) < 3 {
			continue
		}
		// Skip lines that look like labels or markers
		if strings.HasPrefix(line, "(") || strings.HasPrefix(line, "Item") {
			continue
		}
		// This looks like the issuer name
		return line
	}

	return ""
}

// extractBoldBeforeMarker finds text in <B> tags that appear before a marker
// This handles the cover page format where values are in bold before labels
func extractBoldBeforeMarker(doc *html.Node, marker string) string {
	// Find all paragraphs in order
	paragraphs := findAllParagraphsInOrder(doc)

	// Find the paragraph containing the marker
	var markerParagraphIdx = -1
	for i, p := range paragraphs {
		text := extractText(p)
		// Check for marker with and without HTML entities
		if strings.Contains(text, marker) ||
			strings.Contains(text, strings.ReplaceAll(marker, " ", "\u00a0")) || // nbsp
			strings.Contains(text, strings.ReplaceAll(marker, " ", "&nbsp;")) {
			markerParagraphIdx = i
			break
		}
	}

	if markerParagraphIdx == -1 {
		return ""
	}

	// Look backwards through previous paragraphs for one with meaningful text
	for i := markerParagraphIdx - 1; i >= 0 && i >= markerParagraphIdx-5; i-- {
		// First try to find <B> tag (most common)
		boldText := findFirstBoldInNode(paragraphs[i])
		if boldText != "" && len(boldText) > 2 {
			return boldText
		}

		// If no <B> tag, extract the full paragraph text
		// (handles cases where value is in <FONT> or other tags)
		paraText := extractText(paragraphs[i])
		paraText = strings.TrimSpace(paraText)
		// Clean up whitespace and remove nbsp
		re := regexp.MustCompile(`\s+`)
		paraText = re.ReplaceAllString(paraText, " ")

		// Skip empty paragraphs and nbsp-only paragraphs
		if paraText == "" || paraText == " " || strings.TrimSpace(strings.ReplaceAll(paraText, "\u00a0", "")) == "" {
			continue
		}

		// Skip paragraphs that look like labels (start with parentheses)
		if strings.HasPrefix(paraText, "(") {
			continue
		}

		// This looks like a value - return it
		if len(paraText) > 2 {
			return paraText
		}
	}

	return ""
}

// findFirstBoldInNode finds the first <B> tag text in a node
func findFirstBoldInNode(n *html.Node) string {
	var result string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != "" {
			return // Already found
		}
		if n.Type == html.ElementNode && n.Data == "b" {
			text := extractText(n)
			result = strings.TrimSpace(text)
			// Clean up whitespace
			re := regexp.MustCompile(`\s+`)
			result = re.ReplaceAllString(result, " ")
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
}

// findAllParagraphsInOrder finds all <P> tags in document order
func findAllParagraphsInOrder(n *html.Node) []*html.Node {
	var paras []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			paras = append(paras, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return paras
}

// findAllBoldTexts finds all text content within <B> tags
func findAllBoldTexts(n *html.Node) []string {
	var texts []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "b" {
			text := extractText(n)
			text = strings.TrimSpace(text)
			// Clean up whitespace
			re := regexp.MustCompile(`\s+`)
			text = re.ReplaceAllString(text, " ")
			if text != "" {
				texts = append(texts, text)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return texts
}

// cleanReportingPersonName removes trailing row numbers and extra whitespace
// Examples: "Baker Bros. Advisors LP    2." -> "Baker Bros. Advisors LP"
func cleanReportingPersonName(name string) string {
	// Remove trailing numbers with periods (row numbers like "2.", "3.", etc.)
	re := regexp.MustCompile(`\s+\d+\.\s*$`)
	name = re.ReplaceAllString(name, "")

	// Clean up excessive whitespace
	re = regexp.MustCompile(`\s+`)
	name = re.ReplaceAllString(name, " ")

	return strings.TrimSpace(name)
}

// extractFieldBeforeMarker finds the value before a marker like "(Name of Issuer)"
// DEPRECATED: Use extractBoldBeforeMarker or extractFromItem1a instead
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

// extractSchedule13DItems extracts narrative Items 1-7 from Schedule 13D HTML
func extractSchedule13DItems(doc *html.Node) *Schedule13DItems {
	items := &Schedule13DItems{}

	// Find Item paragraphs by looking for bold "Item N" headings in the DOM
	itemParas := findItemParagraphs(doc)

	// Extract content between Item paragraphs
	items.Item1SecurityTitle = extractItemContentDOM(doc, itemParas, 1)
	items.Item2FilingPersons = extractItemContentDOM(doc, itemParas, 2)
	items.Item3SourceOfFunds = extractItemContentDOM(doc, itemParas, 3)
	items.Item4PurposeOfTransaction = extractItemContentDOM(doc, itemParas, 4)
	items.Item5PercentageOfClass = extractItemContentDOM(doc, itemParas, 5)
	items.Item6Contracts = extractItemContentDOM(doc, itemParas, 6)
	items.Item7Exhibits = extractItemContentDOM(doc, itemParas, 7)

	// Clean up extracted text
	items.Item1SecurityTitle = cleanItemText(items.Item1SecurityTitle)
	items.Item2FilingPersons = cleanItemText(items.Item2FilingPersons)
	items.Item3SourceOfFunds = cleanItemText(items.Item3SourceOfFunds)
	items.Item4PurposeOfTransaction = cleanItemText(items.Item4PurposeOfTransaction)
	items.Item5PercentageOfClass = cleanItemText(items.Item5PercentageOfClass)
	items.Item6Contracts = cleanItemText(items.Item6Contracts)
	items.Item7Exhibits = cleanItemText(items.Item7Exhibits)

	return items
}

// extractSchedule13GItems extracts narrative Items 1-10 from Schedule 13G HTML
func extractSchedule13GItems(doc *html.Node) *Schedule13GItems {
	items := &Schedule13GItems{}
	pageText := extractText(doc)

	// Extract each item by finding text between Item markers
	items.Item1IssuerName = extractItemText(pageText, "Item 1", "Item 2")
	items.Item2FilerNames = extractItemText(pageText, "Item 2", "Item 3")
	// Item 3 is usually "Not Applicable"
	items.Item3NotApplicable = strings.Contains(extractItemText(pageText, "Item 3", "Item 4"), "Not Applicable")

	item4Text := extractItemText(pageText, "Item 4", "Item 5")
	items.Item4AmountBeneficiallyOwned = cleanItemText(item4Text)

	item5Text := extractItemText(pageText, "Item 5", "Item 6")
	items.Item5NotApplicable = strings.Contains(item5Text, "Not Applicable")
	items.Item5Ownership5PctOrLess = cleanItemText(item5Text)

	item6Text := extractItemText(pageText, "Item 6", "Item 7")
	items.Item6NotApplicable = strings.Contains(item6Text, "Not Applicable")

	item7Text := extractItemText(pageText, "Item 7", "Item 8")
	items.Item7NotApplicable = strings.Contains(item7Text, "Not Applicable")

	item8Text := extractItemText(pageText, "Item 8", "Item 9")
	items.Item8NotApplicable = strings.Contains(item8Text, "Not Applicable")

	item9Text := extractItemText(pageText, "Item 9", "Item 10")
	items.Item9NotApplicable = strings.Contains(item9Text, "Not Applicable")

	item10Text := extractItemText(pageText, "Item 10", "SIGNATURE")
	items.Item10Certification = cleanItemText(item10Text)

	return items
}

// extractItemText extracts text between two markers (e.g., between "Item 4" and "Item 5")
func extractItemText(text, startMarker, endMarker string) string {
	// Find the start marker - look for the Item heading with bold formatting
	// Pattern: "Item N." or "Item N ." where N is the number
	startPattern := startMarker
	if !strings.HasSuffix(startPattern, ".") {
		startPattern += "."
	}

	startIdx := strings.Index(text, startPattern)
	if startIdx == -1 {
		// Try with space before period: "Item 4 ."
		startPattern = strings.TrimSuffix(startMarker, ".") + " ."
		startIdx = strings.Index(text, startPattern)
		if startIdx == -1 {
			return ""
		}
	}

	// Skip past the item heading line (everything up to next paragraph)
	// Look for double newline or "Purpose of Transaction" or other title text
	start := startIdx
	// Find end of the Item title line (look for first paragraph break)
	searchArea := text[start : start+min(500, len(text)-start)]

	// Skip the Item N. heading and its title (e.g., "Purpose of Transaction")
	// Look for the first real paragraph (after the heading)
	titleEndIdx := strings.Index(searchArea, "\n\n")
	if titleEndIdx == -1 {
		// If no double newline, skip past first line and some padding
		titleEndIdx = strings.Index(searchArea, "\n")
		if titleEndIdx != -1 {
			titleEndIdx += 1
		} else {
			titleEndIdx = len(startPattern) + 50 // Default skip
		}
	}
	start += titleEndIdx

	// Find the end marker (next Item)
	endIdx := strings.Index(text[start:], endMarker)
	if endIdx == -1 {
		// Take rest of text up to reasonable limit
		endIdx = len(text) - start
		if endIdx > 50000 {
			endIdx = 50000
		}
	}

	extracted := text[start : start+endIdx]

	// Remove any remaining Item heading fragments at the start
	lines := strings.Split(extracted, "\n")
	var contentLines []string
	skipLines := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip first few lines if they look like headings
		if i < 3 && (trimmed == "" ||
			strings.Contains(trimmed, "Purpose") ||
			strings.Contains(trimmed, "Transaction") ||
			strings.Contains(trimmed, "Identity") ||
			strings.Contains(trimmed, "Background") ||
			len(trimmed) < 10) {
			skipLines++
			continue
		}
		contentLines = append(contentLines, line)
	}

	return strings.Join(contentLines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// cleanItemText cleans up extracted item text
func cleanItemText(text string) string {
	// Trim whitespace
	text = strings.TrimSpace(text)

	// Collapse multiple whitespace/newlines into single space
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Remove page markers
	text = regexp.MustCompile(`Page \d+ of \d+`).ReplaceAllString(text, "")

	// Trim again
	text = strings.TrimSpace(text)

	return text
}

// findItemSectionPositions finds the byte positions of Item section headings
// Returns a map of item number -> start position in text
// Uses regex to distinguish section headings from inline text references
func findItemSectionPositions(text string) map[int]int {
	positions := make(map[int]int)

	// Pattern: "Item" followed by spaces/nbsp and a number and period
	// Preceded by significant whitespace or start of text (to avoid "see Item 4 below")
	// Handles formats like "Item 4." "Item  4." "Item   4 ." etc.
	// The (?:^|\s{3,}|\.\s+) requires Item to be preceded by start, 3+ spaces, or period+space
	pattern := regexp.MustCompile(`(?:^|\s{3,}|\.\s+)Item\s+(\d+)\s*\.`)

	matches := pattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			itemNumStr := text[match[2]:match[3]]
			itemNum, err := strconv.Atoi(itemNumStr)
			if err == nil && itemNum >= 1 && itemNum <= 10 {
				// Adjust position to start of "Item" word (skip the preceding whitespace/period)
				itemStart := match[0]
				// Find where "Item" actually starts
				searchText := text[itemStart:match[1]]
				if idx := strings.Index(searchText, "Item"); idx != -1 {
					itemStart += idx
				}
				positions[itemNum] = itemStart
			}
		}
	}

	return positions
}

// extractItemByNumber extracts an Item's content using the position map
func extractItemByNumber(text string, positions map[int]int, itemNum int) string {
	startPos, ok := positions[itemNum]
	if !ok {
		return ""
	}

	// Find end position (start of next Item, or end of text)
	endPos := len(text)
	for nextNum := itemNum + 1; nextNum <= 11; nextNum++ {
		if nextPos, ok := positions[nextNum]; ok {
			endPos = nextPos
			break
		}
	}

	// Special handling: if no next Item found, look for SIGNATURE
	if endPos == len(text) {
		if sigIdx := strings.Index(text[startPos:], "SIGNATURE"); sigIdx != -1 {
			endPos = startPos + sigIdx
		}
	}

	// Extract the text
	itemText := text[startPos:endPos]

	// Skip the Item heading line (first line)
	if firstNewline := strings.Index(itemText, "\n"); firstNewline != -1 {
		itemText = itemText[firstNewline+1:]
	}

	// Skip additional heading/title lines (look for first real paragraph)
	lines := strings.Split(itemText, "\n")
	var contentLines []string
	foundContent := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and short lines at the start (likely titles)
		if !foundContent && (trimmed == "" || len(trimmed) < 15) {
			continue
		}

		// Once we find real content, keep everything
		foundContent = true
		contentLines = append(contentLines, line)
	}

	return strings.Join(contentLines, "\n")
}

// findItemParagraphs finds all paragraphs that contain Item headings
// Returns a map of item number -> paragraph node
func findItemParagraphs(doc *html.Node) map[int]*html.Node {
	itemParas := make(map[int]*html.Node)

	// Get all paragraphs in order
	paras := findAllParagraphsInOrder(doc)

	// Pattern to match "Item N." in text (handles "Item  4." "Item   4 ." etc.)
	itemPattern := regexp.MustCompile(`Item\s+(\d+)\s*\.`)

	for _, para := range paras {
		// Check if this paragraph contains bold text with "Item N."
		paraText := extractText(para)

		// Only consider paragraphs that contain "Item" at the start
		trimmed := strings.TrimSpace(paraText)
		if !strings.HasPrefix(trimmed, "Item") {
			continue
		}

		// For Item headings, check first 300 chars (enough for "Item N. Long Title Name")
		searchText := trimmed
		if len(searchText) > 300 {
			searchText = searchText[:300]
		}

		// Normalize non-breaking spaces to regular spaces (Go's \s doesn't match \u00a0)
		searchText = strings.ReplaceAll(searchText, "\u00a0", " ")

		// Check if it matches the Item pattern
		if matches := itemPattern.FindStringSubmatch(searchText); len(matches) >= 2 {
			itemNum, err := strconv.Atoi(matches[1])
			if err == nil && itemNum >= 1 && itemNum <= 10 {
				itemParas[itemNum] = para
			}
		}
	}

	return itemParas
}

// extractItemContentDOM extracts content between two Item paragraph nodes
func extractItemContentDOM(doc *html.Node, itemParas map[int]*html.Node, itemNum int) string {
	startPara, ok := itemParas[itemNum]
	if !ok {
		return ""
	}

	// Find the next Item paragraph (any Item number greater than this one)
	var endPara *html.Node
	for nextNum := itemNum + 1; nextNum <= 11; nextNum++ {
		if p, ok := itemParas[nextNum]; ok {
			endPara = p
			break
		}
	}

	// Extract all paragraphs between start and end
	allParas := findAllParagraphsInOrder(doc)
	var contentParas []*html.Node
	capturing := false

	for _, para := range allParas {
		if para == startPara {
			capturing = true
			continue // Skip the Item heading itself
		}
		if endPara != nil && para == endPara {
			break
		}
		if capturing {
			contentParas = append(contentParas, para)
		}
	}

	// If no end found, look for SIGNATURE
	if endPara == nil && len(contentParas) > 0 {
		// Stop at SIGNATURE or reasonable limit
		var finalParas []*html.Node
		for _, para := range contentParas {
			paraText := extractText(para)
			if strings.Contains(paraText, "SIGNATURE") {
				break
			}
			finalParas = append(finalParas, para)
			if len(finalParas) >= 500 { // Safety limit
				break
			}
		}
		contentParas = finalParas
	}

	// Combine all paragraph texts
	var textParts []string
	for _, para := range contentParas {
		paraText := extractText(para)
		paraText = strings.TrimSpace(paraText)
		if paraText != "" {
			textParts = append(textParts, paraText)
		}
	}

	return strings.Join(textParts, " ")
}
