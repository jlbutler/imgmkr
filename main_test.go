package main

import (
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"512KB", 512 * KB, false},
		{"1MB", 1 * MB, false},
		{"2GB", 2 * GB, false},
		{"1.5MB", int64(1.5 * MB), false},
		{"2.75GB", int64(2.75 * GB), false},
		{"1024", 1024, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"1.5XB", 0, true},
	}

	for _, test := range tests {
		result, err := parseSize(test.input)

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

func TestParseSizes(t *testing.T) {
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
		result, err := parseSizes(test.input)

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

func TestFormatSize(t *testing.T) {
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
		result := formatSize(test.input)
		if result != test.expected {
			t.Errorf("For input %d, expected %q, got %q", test.input, test.expected, result)
		}
	}
}
