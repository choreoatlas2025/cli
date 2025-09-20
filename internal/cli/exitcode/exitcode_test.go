package exitcode

import "testing"

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"OK should be 0", OK, 0},
		{"CLIError should be 1", CLIError, 1},
		{"InputError should be 2", InputError, 2},
		{"ValidationFailed should be 3", ValidationFailed, 3},
		{"GateFailed should be 4", GateFailed, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, tt.code)
			}
		})
	}
}

func TestExitCodeUniqueness(t *testing.T) {
	codes := map[int]string{
		OK:               "OK",
		CLIError:         "CLIError",
		InputError:       "InputError",
		ValidationFailed: "ValidationFailed",
		GateFailed:       "GateFailed",
	}

	// Check all codes are unique
	seen := make(map[int]bool)
	for code, name := range codes {
		if seen[code] {
			t.Errorf("Duplicate exit code %d for %s", code, name)
		}
		seen[code] = true
	}
}