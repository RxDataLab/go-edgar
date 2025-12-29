package edgar

import (
	"testing"
)

func TestExtract10b51(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		expectedIs10b51 bool
		expectedDate    *string
	}{
		{
			name:            "No 10b5-1 mention",
			text:            "This is a regular footnote about something else.",
			expectedIs10b51: false,
			expectedDate:    nil,
		},
		{
			name:            "10b5-1 with date - March 13, 2025",
			text:            "The sales reported in this Form 4 were effected pursuant to a Rule 10b5-1 trading plan adopted by the reporting person on March 13, 2025.",
			expectedIs10b51: true,
			expectedDate:    stringPtr("2025-03-13"),
		},
		{
			name:            "10b5-1 with date - September 18, 2025",
			text:            "Reported transaction occurred pursuant to a Rule 10b5-1 Plan adopted by the reporting person on September 18, 2025.",
			expectedIs10b51: true,
			expectedDate:    stringPtr("2025-09-18"),
		},
		{
			name:            "10b5-1 without date",
			text:            "Shares were sold pursuant to a 10b5-1 trading plan adopted by the Reporting Person in accordance with Rule 10b5-1 of the Securities Exchange Act of 1934, as amended.",
			expectedIs10b51: true,
			expectedDate:    nil,
		},
		{
			name:            "10b5-1 without positive language (should not match)",
			text:            "The 10b5-1 plan was terminated on March 13, 2025.",
			expectedIs10b51: false,
			expectedDate:    nil,
		},
		{
			name:            "10b5-1 with hyphen variation",
			text:            "Pursuant to Rule 10b5–1 trading plan adopted on January 5, 2024.",
			expectedIs10b51: true,
			expectedDate:    stringPtr("2024-01-05"),
		},
		{
			name:            "Arrowhead case - in accordance with",
			text:            "Shares were sold pursuant to a 10b5-1 trading plan adopted by the Reporting Person in accordance with Rule 10b5-1 of the Securities Exchange Act of 1934, as amended.",
			expectedIs10b51: true,
			expectedDate:    nil,
		},
		{
			name:            "Date with single digit day",
			text:            "Trading plan adopted pursuant to Rule 10b5-1 on May 5, 2024.",
			expectedIs10b51: true,
			expectedDate:    stringPtr("2024-05-05"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Extract10b51(tt.text)

			if result.Is10b51Plan != tt.expectedIs10b51 {
				t.Errorf("Is10b51Plan = %v, want %v", result.Is10b51Plan, tt.expectedIs10b51)
			}

			// Compare dates
			if tt.expectedDate == nil {
				if result.TenB51AdoptionDate != nil {
					t.Errorf("TenB51AdoptionDate = %v, want nil", *result.TenB51AdoptionDate)
				}
			} else {
				if result.TenB51AdoptionDate == nil {
					t.Errorf("TenB51AdoptionDate = nil, want %v", *tt.expectedDate)
				} else if *result.TenB51AdoptionDate != *tt.expectedDate {
					t.Errorf("TenB51AdoptionDate = %v, want %v", *result.TenB51AdoptionDate, *tt.expectedDate)
				}
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected *string
	}{
		{"March 13, 2025", stringPtr("2025-03-13")},
		{"September 18, 2025", stringPtr("2025-09-18")},
		{"Jan 5, 2024", stringPtr("2024-01-05")},
		{"May 5, 2024", stringPtr("2024-05-05")},
		{"December 1, 2023", stringPtr("2023-12-01")},
		{"Invalid date", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDate(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("parseDate(%q) = %v, want nil", tt.input, *result)
				}
			} else {
				if result == nil {
					t.Errorf("parseDate(%q) = nil, want %v", tt.input, *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("parseDate(%q) = %v, want %v", tt.input, *result, *tt.expected)
				}
			}
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
