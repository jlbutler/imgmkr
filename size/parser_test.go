package size

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		// Bytes variations
		{"8150", 8150, false},
		{"8B", 8, false},
		{"8b", 8, false},
		{"8byte", 8, false},
		{"8bytes", 8, false},
		{"8BYTES", 8, false},

		// Kilobytes variations
		{"512KB", 512 * KB, false},
		{"512kb", 512 * KB, false},
		{"512K", 512 * KB, false},
		{"512k", 512 * KB, false},
		{"1.5K", int64(1.5 * KB), false},

		// Megabytes variations
		{"1MB", 1 * MB, false},
		{"1mb", 1 * MB, false},
		{"1M", 1 * MB, false},
		{"1m", 1 * MB, false},
		{"1.5MB", int64(1.5 * MB), false},
		{"2.75m", int64(2.75 * MB), false},

		// Gigabytes variations
		{"2GB", 2 * GB, false},
		{"2gb", 2 * GB, false},
		{"2G", 2 * GB, false},
		{"2g", 2 * GB, false},
		{"2.75GB", int64(2.75 * GB), false},
		{"1.5G", int64(1.5 * GB), false},

		// Edge cases and errors
		{"1024", 1024, false},
		{"0", 0, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"1.5XB", 0, true},
		{"MB", 0, true},
		{"1.2.3MB", 0, true},
	}

	for _, test := range tests {
		result, err := Parse(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %q, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("For input %q, expected %d, got %d", test.input, test.expected, result)
			}
		}
	}
}

func TestParseList(t *testing.T) {
	tests := []struct {
		input    string
		expected []int64
		hasError bool
	}{
		{"512KB,1MB,2GB", []int64{512 * KB, 1 * MB, 2 * GB}, false},
		{"1MB", []int64{1 * MB}, false},
		{"", nil, true},
		{"1MB,invalid", nil, true},
	}

	for _, test := range tests {
		result, err := ParseList(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %q, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", test.input, err)
			}
			if len(result) != len(test.expected) {
				t.Errorf("For input %q, expected length %d, got %d", test.input, len(test.expected), len(result))
				continue
			}
			for i, expected := range test.expected {
				if result[i] != expected {
					t.Errorf("For input %q at index %d, expected %d, got %d", test.input, i, expected, result[i])
				}
			}
		}
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{512, "512 bytes"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{int64(1.5 * MB), "1.50 MB"},
		{int64(2.75 * GB), "2.75 GB"},
	}

	for _, test := range tests {
		result := Format(test.input)
		if result != test.expected {
			t.Errorf("For input %d, expected %q, got %q", test.input, test.expected, result)
		}
	}
}
