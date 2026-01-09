package edgar

import (
	"os"
	"testing"
)

func TestDetectFormType(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		want     string
	}{
		{
			name:     "Schedule 13D",
			filepath: "/home/nick/projects/port-edgartools/edgartools/tests/data/beneficial_ownership/schedule13d.xml",
			want:     "SCHEDULE 13D",
		},
		{
			name:     "Schedule 13G",
			filepath: "/home/nick/projects/port-edgartools/edgartools/tests/data/beneficial_ownership/schedule13g.xml",
			want:     "SCHEDULE 13G",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.filepath)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			got, err := detectFormType(data)
			if err != nil {
				t.Fatalf("detectFormType() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("detectFormType() = %v, want %v", got, tt.want)
			}
		})
	}
}
