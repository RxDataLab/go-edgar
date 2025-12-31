package edgar

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// ParseInlineXBRL parses an inline XBRL (iXBRL) document from HTML
// Inline XBRL embeds XBRL facts within HTML using the ix: namespace
func ParseInlineXBRL(data []byte) (*XBRL, error) {
	xbrl := &XBRL{}

	// Parse contexts and units from ix:resources section
	if err := extractResources(xbrl, data); err != nil {
		return nil, fmt.Errorf("failed to extract resources: %w", err)
	}

	// Extract facts from ix:nonFraction and ix:nonNumeric tags
	if err := extractInlineFacts(xbrl, data); err != nil {
		return nil, fmt.Errorf("failed to extract facts: %w", err)
	}

	// Resolve contexts and standardize labels
	if err := resolveFacts(xbrl); err != nil {
		return nil, fmt.Errorf("failed to resolve facts: %w", err)
	}

	return xbrl, nil
}

// extractResources extracts contexts and units from the ix:resources section
func extractResources(xbrl *XBRL, data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		// Treat ASCII and other charsets as UTF-8
		return input, nil
	}

	inResources := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			// Track when we enter/exit ix:resources
			if elem.Name.Local == "resources" {
				inResources = true
				continue
			}

			if !inResources {
				continue
			}

			// Parse context elements
			if elem.Name.Local == "context" {
				var ctx Context
				if err := decoder.DecodeElement(&ctx, &elem); err != nil {
					continue // Skip malformed contexts
				}
				xbrl.Contexts = append(xbrl.Contexts, ctx)
			}

			// Parse unit elements
			if elem.Name.Local == "unit" {
				var unit Unit
				if err := decoder.DecodeElement(&unit, &elem); err != nil {
					continue // Skip malformed units
				}
				xbrl.Units = append(xbrl.Units, unit)
			}

		case xml.EndElement:
			if elem.Name.Local == "resources" {
				inResources = false
			}
		}
	}

	return nil
}

// extractInlineFacts extracts facts from ix:nonFraction and ix:nonNumeric tags
func extractInlineFacts(xbrl *XBRL, data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		// Treat ASCII and other charsets as UTF-8
		return input, nil
	}

	var facts []Fact

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			// Check for inline XBRL fact elements (ix:nonFraction, ix:nonNumeric)
			if elem.Name.Local != "nonFraction" && elem.Name.Local != "nonNumeric" {
				continue
			}

			// Extract attributes
			contextRef := getAttr(elem.Attr, "contextRef")
			if contextRef == "" {
				continue // Not a valid fact
			}

			conceptName := getAttr(elem.Attr, "name")
			if conceptName == "" {
				continue // No concept name
			}

			unitRef := getAttr(elem.Attr, "unitRef")
			decimalsStr := getAttr(elem.Attr, "decimals")

			// Parse decimals
			decimals := 0
			if decimalsStr != "" && decimalsStr != "INF" {
				fmt.Sscanf(decimalsStr, "%d", &decimals)
			}

			// Extract the fact value (text content)
			var value string
			if err := decoder.DecodeElement(&value, &elem); err != nil {
				continue
			}

			fact := Fact{
				Concept:    conceptName,
				Value:      strings.TrimSpace(value),
				ContextRef: contextRef,
				UnitRef:    unitRef,
				Decimals:   decimals,
			}

			facts = append(facts, fact)
		}
	}

	xbrl.Facts = facts
	return nil
}

// DetectXBRLType determines if the data is inline XBRL or standalone XBRL
func DetectXBRLType(data []byte) string {
	content := string(data)

	// Check for inline XBRL markers
	if strings.Contains(content, "xmlns:ix=") ||
		strings.Contains(content, "<ix:") ||
		strings.Contains(content, "inlineXBRL") {
		return "inline"
	}

	// Check for standalone XBRL markers
	if strings.Contains(content, "<xbrl") ||
		strings.Contains(content, "xmlns:xbrli=") {
		return "standalone"
	}

	return "unknown"
}

// ParseXBRLAuto automatically detects and parses inline or standalone XBRL
func ParseXBRLAuto(data []byte) (*XBRL, error) {
	xbrlType := DetectXBRLType(data)

	switch xbrlType {
	case "inline":
		return ParseInlineXBRL(data)
	case "standalone":
		return ParseXBRL(data)
	default:
		return nil, fmt.Errorf("unable to detect XBRL type")
	}
}
