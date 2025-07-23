package utils

import (
	"testing"
	"encoding/base64" // Added this import
)

// TestRandString_PositiveLength tests RandString with a positive length.
func TestRandString_PositiveLength(t *testing.T) {
	length := 10
	s, err := RandString(length)
	if err != nil {
		t.Fatalf("RandString(%d) returned an unexpected error: %v", length, err)
	}
	if s == "" {
		t.Errorf("RandString(%d) returned an empty string", length)
	}
	expectedEncodedLen := base64.RawURLEncoding.EncodedLen(length)
	if len(s) != expectedEncodedLen {
		t.Errorf("RandString(%d) returned string of length %d, expected %d", length, len(s), expectedEncodedLen)
	}
}

// TestRandString_ZeroLength tests RandString with a zero length.
func TestRandString_ZeroLength(t *testing.T) {
	length := 0
	s, err := RandString(length)
	if err != nil {
		t.Fatalf("RandString(%d) returned an unexpected error: %v", length, err)
	}
	if s != "" {
		t.Errorf("RandString(%d) returned non-empty string '%s', expected empty", length, s)
	}
	expectedEncodedLen := base64.RawURLEncoding.EncodedLen(length)
	if len(s) != expectedEncodedLen {
		t.Errorf("RandString(%d) returned string of length %d, expected %d", length, len(s), expectedEncodedLen)
	}
}

// TestContainsIgnoreCase_ExactMatch tests ContainsIgnoreCase with an exact match.
func TestContainsIgnoreCase_ExactMatch(t *testing.T) {
	fullstring := "Hello World"
	searchfor := "World"
	expected := true
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_CaseInsensitiveMatch tests ContainsIgnoreCase with a case-insensitive match.
func TestContainsIgnoreCase_CaseInsensitiveMatch(t *testing.T) {
	fullstring := "Hello World"
	searchfor := "world"
	expected := true
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_MixedCaseMatch tests ContainsIgnoreCase with mixed case input.
func TestContainsIgnoreCase_MixedCaseMatch(t *testing.T) {
	fullstring := "Hello World"
	searchfor := "WoRlD"
	expected := true
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_SubstringNotFound tests ContainsIgnoreCase when the substring is not present.
func TestContainsIgnoreCase_SubstringNotFound(t *testing.T) {
	fullstring := "Hello World"
	searchfor := "Goodbye"
	expected := false
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_EmptySearchForString tests ContainsIgnoreCase with an empty search string.
func TestContainsIgnoreCase_EmptySearchForString(t *testing.T) {
	fullstring := "Hello World"
	searchfor := ""
	expected := true // strings.Contains("abc", "") is true
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_EmptyFullString tests ContainsIgnoreCase with an empty full string.
func TestContainsIgnoreCase_EmptyFullString(t *testing.T) {
	fullstring := ""
	searchfor := "test"
	expected := false
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_BothEmpty tests ContainsIgnoreCase when both strings are empty.
func TestContainsIgnoreCase_BothEmpty(t *testing.T) {
	fullstring := ""
	searchfor := ""
	expected := true
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_FullStringShorter tests ContainsIgnoreCase when the full string is shorter than the search string.
func TestContainsIgnoreCase_FullStringShorter(t *testing.T) {
	fullstring := "abc"
	searchfor := "abcdef"
	expected := false
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_MultipleOccurrences tests ContainsIgnoreCase when the search string appears multiple times.
func TestContainsIgnoreCase_MultipleOccurrences(t *testing.T) {
	fullstring := "banana republic"
	searchfor := "ana"
	expected := true
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_AtStart tests ContainsIgnoreCase when the search string is at the beginning.
func TestContainsIgnoreCase_AtStart(t *testing.T) {
	fullstring := "Apple Pie"
	searchfor := "apple"
	expected := true
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}

// TestContainsIgnoreCase_AtEnd tests ContainsIgnoreCase when the search string is at the end.
func TestContainsIgnoreCase_AtEnd(t *testing.T) {
	fullstring := "Apple Pie"
	searchfor := "pie"
	expected := true
	result := ContainsIgnoreCase(fullstring, searchfor)
	if result != expected {
		t.Errorf("ContainsIgnoreCase(\"%s\", \"%s\") = %v, want %v", fullstring, searchfor, result, expected)
	}
}
