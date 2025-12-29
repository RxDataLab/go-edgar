package edgar

import (
	"regexp"
	"strings"
	"time"
)

// TenB51Result represents the result of analyzing text for 10b5-1 plan information
type TenB51Result struct {
	Is10b51Plan        bool
	TenB51AdoptionDate *string // ISO-8601 format (YYYY-MM-DD), nil if not found
}

var (
	// Detect 10b5-1 plan references (various formats: 10b5-1, 10b5–1, Rule 10b5-1, etc.)
	re10b51 = regexp.MustCompile(`(?i)\b(rule\s*)?10b5[-–]?1\b`)

	// Positive language indicating active plan usage (not cancellation/termination)
	rePositive = regexp.MustCompile(`(?i)\b(pursuant\s+to|adopted|in\s+accordance\s+with|under|effected\s+pursuant\s+to)\b`)

	// Date extraction near adoption language
	// Captures dates like "on March 13, 2025" or "in September 2025"
	reAdoptionDate = regexp.MustCompile(
		`(?i)\b(adopted|established|entered\s+into).*?\b(on|in)\s+` +
			`((?:January|February|March|April|May|June|July|August|September|October|November|December|` +
			`Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Sept|Oct|Nov|Dec)` +
			`\s+\d{1,2},\s+\d{4}|` + // "March 13, 2025"
			`(?:January|February|March|April|May|June|July|August|September|October|November|December|` +
			`Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Sept|Oct|Nov|Dec)` +
			`\s+\d{4})`, // "March 2025"
	)
)

// parseDate tries multiple date layouts and returns ISO-8601 format (YYYY-MM-DD)
// Returns nil if parsing fails
func parseDate(raw string) *string {
	raw = strings.TrimSpace(raw)

	layouts := []string{
		"January 2, 2006", // Full month name with day
		"Jan 2, 2006",     // Abbreviated month with day
		"January, 2006",   // Full month name, year only (with comma)
		"Jan, 2006",       // Abbreviated month, year only (with comma)
		"January 2006",    // Full month name, year (no comma)
		"Jan 2006",        // Abbreviated month, year (no comma)
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			iso := t.Format("2006-01-02")
			return &iso
		}
	}

	return nil
}

// Extract10b51 analyzes text (typically a footnote) for 10b5-1 plan information
// Returns whether it's a 10b5-1 plan transaction and the adoption date if found
func Extract10b51(text string) TenB51Result {
	result := TenB51Result{}

	// Step 1: Check if text mentions 10b5-1
	if !re10b51.MatchString(text) {
		return result
	}

	// Step 2: Check for positive language (not a cancellation/termination)
	// If no positive language, don't treat as a plan transaction
	if !rePositive.MatchString(text) {
		return result
	}

	result.Is10b51Plan = true

	// Step 3: Attempt to extract adoption date
	match := reAdoptionDate.FindStringSubmatch(text)
	if len(match) >= 4 {
		// match[3] contains the date portion
		if date := parseDate(match[3]); date != nil {
			result.TenB51AdoptionDate = date
		}
	}

	return result
}

// Parse10b51Footnotes analyzes all footnotes AND remarks and returns a map of footnote IDs
// to their adoption dates (in ISO format). Only includes footnotes that indicate
// active 10b5-1 plan usage.
//
// Special case: If remarks contain 10b5-1 info, the map includes key "__REMARKS__"
// which should be applied to transactions that don't have specific footnote references.
//
// Returns: map[footnoteID]adoptionDate (empty string if no date found but is 10b5-1)
func (f *Form4) Parse10b51Footnotes() map[string]string {
	result := make(map[string]string)

	// Check all footnotes
	for _, fn := range f.Footnotes {
		analysis := Extract10b51(fn.Text)
		if analysis.Is10b51Plan {
			if analysis.TenB51AdoptionDate != nil {
				result[fn.ID] = *analysis.TenB51AdoptionDate
			} else {
				// 10b5-1 plan but no date found
				result[fn.ID] = ""
			}
		}
	}

	// Check remarks field (fallback for cases where 10b5-1 is mentioned in remarks, not footnotes)
	if f.Remarks != "" {
		analysis := Extract10b51(f.Remarks)
		if analysis.Is10b51Plan {
			if analysis.TenB51AdoptionDate != nil {
				result["__REMARKS__"] = *analysis.TenB51AdoptionDate
			} else {
				result["__REMARKS__"] = ""
			}
		}
	}

	return result
}
