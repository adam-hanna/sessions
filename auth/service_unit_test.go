// +build unit

package auth

import (
	"reflect"
	"testing"
)

var (
	validKey     = []byte("DOZDgBdMhGLImnk0BGYgOUI+h1n7U+OdxcZPctMbeFCsuAom2aFU4JPV4Qj11hbcb5yaM4WDuNP/3B7b+BnFhw==")
	validService = Service{
		options: Options{
			Key: validKey,
		},
	}
)

// TestNew tests the New function
func TestNew(t *testing.T) {
	var tests = []struct {
		input           Options
		expectedErr     error
		expectedService Service
	}{
		{
			Options{
				Key: validKey,
			},
			nil,
			validService,
		},
		{
			Options{
				Key: []byte{},
			},
			ErrNoSessionKey,
			Service{},
		},
	}

	for idx, tt := range tests {
		var assertService bool

		s, e := New(tt.input)
		if s == nil {
			s = &Service{}
		}

		assertErr := e == tt.expectedErr
		assertService = reflect.DeepEqual(tt.expectedService, *s)

		if !assertErr && !assertService {
			t.Errorf("test #%d failed; assertErr: %t, assertService: %t, expectedErr: %v, expectedService: %v, received err: %v, received service: %v", idx+1, assertErr, assertService, tt.expectedErr, tt.expectedService, *s, e)
		}
	}
}

// TestSignAndBase64Encode tests the SignAndBase64Encode function
func TestSignAndBase64Encode(t *testing.T) {
	// note: err returned is always nil, so don't need to test for it
	var tests = []struct {
		input    string
		expected string
		pass     bool
	}{
		{"5f4cd331-c869-4871-bb41-76b726df9937", "NWY0Y2QzMzEtYzg2OS00ODcxLWJiNDEtNzZiNzI2ZGY5OTM3YGV5KkkGaOaikrAO9qqRa3hocM3OD0JDoXUtJ8LRJKKQw_8H6kAtbps8g4bQHoL--LyxWPesiTvlasxlnnNA7g==", true},
		{"4f4cd331-c869-4871-bb41-76b726df9937", "NWY0Y2QzMzEtYzg2OS00ODcxLWJiNDEtNzZiNzI2ZGY5OTM3YGV5KkkGaOaikrAO9qqRa3hocM3OD0JDoXUtJ8LRJKKQw_8H6kAtbps8g4bQHoL--LyxWPesiTvlasxlnnNA7g=a", false},
	}

	for idx, tt := range tests {
		a, _ := validService.SignAndBase64Encode(tt.input)

		if tt.pass && a != tt.expected {
			t.Errorf("test #%d failed; input: %s, expected output: %s, received: %s", idx+1, tt.input, tt.expected, a)
		} else if !tt.pass && a == tt.expected {
			t.Errorf("test #%d failed; input: %s, expected output: %s, received: %s", idx+1, tt.input, tt.expected, a)
		}
	}
}

// TestSignAndBase64Encode tests the SignAndBase64Encode function
func TestVerifyAndDecode(t *testing.T) {
	// note: err returned is always nil, so don't need to test for it
	var tests = []struct {
		expectedString string
		input          string
		expectedErr    error
	}{
		{"5f4cd331-c869-4871-bb41-76b726df9937", "NWY0Y2QzMzEtYzg2OS00ODcxLWJiNDEtNzZiNzI2ZGY5OTM3YGV5KkkGaOaikrAO9qqRa3hocM3OD0JDoXUtJ8LRJKKQw_8H6kAtbps8g4bQHoL--LyxWPesiTvlasxlnnNA7g==", nil},
		{"", "NWY0Y2QzMzEtYzg2OS00ODcxLWJiNDEtNzZiNzI2ZGY5OTM3YGV5KkkGaOaikrAO9qqRa3hocM3OD0JDoXUtJ8LRJKKQw_8H6kAtbps8g4bQHoL--LyxWPesiTvlasxlnnNA7g=a", ErrBase64Decode},
		{"", "5f4cd331-c869-4871-bb41-76b726df9937", ErrInvalidSession},
		{"", "NAY0Y2QzMzEtYzg2OS00ODcxLWJiNDEtNzZiNzI2ZGY5OTM3YGV5KkkGaOaikrAO9qqRa3hocM3OD0JDoXUtJ8LRJKKQw_8H6kAtbps8g4bQHoL--LyxWPesiTvlasxlnnNA7g==", ErrInvalidSession},
	}

	for idx, tt := range tests {
		a, e := validService.VerifyAndDecode(tt.input)

		if a != tt.expectedString || e != tt.expectedErr {
			t.Errorf("test #%d failed; input: %s, expected string: %s, expected err: %v, received string: %s, received err: %v", idx+1, tt.input, tt.expectedString, tt.expectedErr, a, e)
		}
	}
}
