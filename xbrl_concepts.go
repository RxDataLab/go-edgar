package edgar

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed concept_mappings.json
var conceptMappingsJSON []byte

// ConceptMapping represents the structure of concept_mappings.json
type ConceptMapping struct {
	Schema      string                       `json:"$schema"`
	Description string                       `json:"description"`
	Version     string                       `json:"version"`
	Mappings    map[string]ConceptDefinition `json:"mappings"`
}

// ConceptDefinition defines a standardized concept and its XBRL variations
type ConceptDefinition struct {
	Concepts []string `json:"concepts"`
	Notes    string   `json:"notes"`
}

// conceptMapper provides lookup capabilities for XBRL concepts
type conceptMapper struct {
	mappings      map[string]ConceptDefinition // standardized label -> definition
	reverseLookup map[string]string            // XBRL concept -> standardized label
}

var globalMapper *conceptMapper

func init() {
	var err error
	globalMapper, err = loadConceptMappings()
	if err != nil {
		panic(fmt.Sprintf("Failed to load concept mappings: %v", err))
	}
}

// loadConceptMappings parses the embedded JSON and builds lookup tables
func loadConceptMappings() (*conceptMapper, error) {
	var mapping ConceptMapping
	if err := json.Unmarshal(conceptMappingsJSON, &mapping); err != nil {
		return nil, fmt.Errorf("failed to parse concept_mappings.json: %w", err)
	}

	mapper := &conceptMapper{
		mappings:      mapping.Mappings,
		reverseLookup: make(map[string]string),
	}

	// Build reverse lookup: XBRL concept -> standardized label
	for label, def := range mapping.Mappings {
		for _, concept := range def.Concepts {
			mapper.reverseLookup[concept] = label
		}
	}

	return mapper, nil
}

// GetStandardizedLabel returns the standardized label for an XBRL concept
// Returns empty string if no mapping exists
func (m *conceptMapper) GetStandardizedLabel(xbrlConcept string) string {
	// Try exact match first
	if label, ok := m.reverseLookup[xbrlConcept]; ok {
		return label
	}

	// Try case-insensitive match (some filings vary in capitalization)
	for concept, label := range m.reverseLookup {
		if strings.EqualFold(concept, xbrlConcept) {
			return label
		}
	}

	return ""
}

// GetConcepts returns all XBRL concepts that map to a standardized label
func (m *conceptMapper) GetConcepts(standardizedLabel string) ([]string, error) {
	def, ok := m.mappings[standardizedLabel]
	if !ok {
		return nil, fmt.Errorf("unknown standardized label: %s", standardizedLabel)
	}
	return def.Concepts, nil
}

// GetAllStandardizedLabels returns all available standardized labels
func (m *conceptMapper) GetAllStandardizedLabels() []string {
	labels := make([]string, 0, len(m.mappings))
	for label := range m.mappings {
		labels = append(labels, label)
	}
	return labels
}

// HasMapping returns true if the XBRL concept has a standardized mapping
func (m *conceptMapper) HasMapping(xbrlConcept string) bool {
	return m.GetStandardizedLabel(xbrlConcept) != ""
}

// Public interface functions using global mapper

// GetStandardizedLabel returns the standardized label for an XBRL concept
func GetStandardizedLabel(xbrlConcept string) string {
	return globalMapper.GetStandardizedLabel(xbrlConcept)
}

// GetConceptsForLabel returns all XBRL concepts that map to a standardized label
func GetConceptsForLabel(standardizedLabel string) ([]string, error) {
	return globalMapper.GetConcepts(standardizedLabel)
}

// GetAllStandardizedLabels returns all available standardized labels
func GetAllStandardizedLabels() []string {
	return globalMapper.GetAllStandardizedLabels()
}

// HasMapping returns true if the XBRL concept has a standardized mapping
func HasMapping(xbrlConcept string) bool {
	return globalMapper.HasMapping(xbrlConcept)
}
