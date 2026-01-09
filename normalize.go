package edgar

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// NormalizeText normalizes various Unicode and HTML entity issues that appear in SEC filings.
// This should be called early in the parsing pipeline to ensure consistent text handling.
//
// Normalizations performed:
// - HTML entities (&nbsp;, &mdash;, &ldquo;, etc.) → Unicode equivalents
// - Non-breaking spaces (U+00A0) → regular spaces
// - Various Unicode whitespace → regular spaces
// - Zero-width characters → removed
// - Multiple consecutive whitespace → single space
// - Normalize newlines (CRLF → LF)
func NormalizeText(data []byte) []byte {
	text := string(data)

	// 1. HTML entities to Unicode (common in HTML filings)
	text = normalizeHTMLEntities(text)

	// 2. Unicode whitespace normalization
	text = normalizeWhitespace(text)

	// 3. Remove zero-width and invisible characters
	text = removeInvisibleChars(text)

	// 4. Normalize line endings (CRLF → LF)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	return []byte(text)
}

// normalizeHTMLEntities converts common HTML entities to their Unicode equivalents
func normalizeHTMLEntities(text string) string {
	// Common entities found in SEC filings
	replacements := map[string]string{
		"&nbsp;":   " ",      // Non-breaking space
		"&mdash;":  "\u2014", // Em dash
		"&ndash;":  "\u2013", // En dash
		"&ldquo;":  "\u201c", // Left double quote
		"&rdquo;":  "\u201d", // Right double quote
		"&lsquo;":  "\u2018", // Left single quote
		"&rsquo;":  "\u2019", // Right single quote
		"&amp;":    "&",      // Ampersand
		"&lt;":     "<",      // Less than
		"&gt;":     ">",      // Greater than
		"&quot;":   "\"",     // Quote
		"&apos;":   "'",      // Apostrophe
		"&hellip;": "...",    // Ellipsis
		"&bull;":   "\u2022", // Bullet
		"&trade;":  "\u2122", // Trademark
		"&reg;":    "\u00ae", // Registered
		"&copy;":   "\u00a9", // Copyright
		"&sect;":   "\u00a7", // Section sign
		"&para;":   "\u00b6", // Paragraph sign
		"&#160;":   " ",      // Non-breaking space (numeric)
		"&#8211;":  "\u2013", // En dash (numeric)
		"&#8212;":  "\u2014", // Em dash (numeric)
		"&#8220;":  "\u201c", // Left double quote (numeric)
		"&#8221;":  "\u201d", // Right double quote (numeric)
		"&#8217;":  "\u2019", // Right single quote (numeric)
	}

	for entity, replacement := range replacements {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	// Handle numeric entities (&#NNN;) - common pattern
	numericEntityPattern := regexp.MustCompile(`&#(\d+);`)
	text = numericEntityPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the number
		var code int
		if _, err := fmt.Sscanf(match, "&#%d;", &code); err == nil {
			// Convert common codes to their Unicode equivalents
			switch code {
			case 160: // nbsp
				return " "
			case 8211: // en dash
				return "–"
			case 8212: // em dash
				return "—"
			case 8220, 8221: // quotes
				return "\""
			case 8217: // apostrophe
				return "'"
			default:
				// For other codes, try to convert to Unicode rune
				if code < 0x110000 { // Valid Unicode range
					return string(rune(code))
				}
			}
		}
		return match // Leave unchanged if we can't parse
	})

	return text
}

// normalizeWhitespace converts various Unicode whitespace characters to regular spaces
func normalizeWhitespace(text string) string {
	// Replace various Unicode whitespace with regular space
	// U+00A0 (non-breaking space) is the most common issue
	var result strings.Builder
	result.Grow(len(text))

	for _, r := range text {
		switch r {
		case '\u00A0': // Non-breaking space (NBSP)
			result.WriteRune(' ')
		case '\u2000', '\u2001', '\u2002', '\u2003', '\u2004', '\u2005': // En quad, Em quad, etc.
			result.WriteRune(' ')
		case '\u2006', '\u2007', '\u2008', '\u2009', '\u200A': // Figure space, etc.
			result.WriteRune(' ')
		case '\u202F': // Narrow no-break space
			result.WriteRune(' ')
		case '\u205F': // Medium mathematical space
			result.WriteRune(' ')
		case '\u3000': // Ideographic space
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}

// removeInvisibleChars removes zero-width and other invisible characters
func removeInvisibleChars(text string) string {
	var result strings.Builder
	result.Grow(len(text))

	for _, r := range text {
		// Skip zero-width and format characters
		switch r {
		case '\u200B': // Zero-width space
			continue
		case '\u200C': // Zero-width non-joiner
			continue
		case '\u200D': // Zero-width joiner
			continue
		case '\uFEFF': // Zero-width no-break space (BOM)
			continue
		case '\u180E': // Mongolian vowel separator
			continue
		default:
			// Also skip other format characters
			if unicode.Is(unicode.Cf, r) && r != '\t' && r != '\n' && r != '\r' {
				continue
			}
			result.WriteRune(r)
		}
	}

	return result.String()
}

// NormalizeXMLText is a lighter version for XML content that preserves more structure
// but still handles the most common issues
func NormalizeXMLText(data []byte) []byte {
	text := string(data)

	// For XML, we want to be more conservative
	// Only normalize the most problematic characters

	// 1. Convert HTML entities that might appear in XML CDATA
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// 2. Normalize non-breaking spaces
	text = strings.ReplaceAll(text, "\u00A0", " ")

	// 3. Remove zero-width spaces (these should never be in XML)
	text = strings.ReplaceAll(text, "\u200B", "")
	text = strings.ReplaceAll(text, "\uFEFF", "")

	// 4. Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	return []byte(text)
}

// CleanExtractedText is for cleaning text AFTER extraction from parsed documents
// This is more aggressive than input normalization
func CleanExtractedText(text string) string {
	// Collapse multiple whitespace into single space
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Remove page markers
	text = regexp.MustCompile(`Page \d+ of \d+`).ReplaceAllString(text, "")

	// Trim
	text = strings.TrimSpace(text)

	return text
}
