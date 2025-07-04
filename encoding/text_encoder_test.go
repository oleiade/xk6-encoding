package encoding

import (
	"bytes"
	"testing"
)

func TestNewTextEncoder(t *testing.T) {
	encoder := NewTextEncoder()

	if encoder == nil {
		t.Fatal("NewTextEncoder() should not return nil")
	}

	if encoder.Encoding != UTF8EncodingFormat {
		t.Errorf("Expected encoding to be %s, got %s", UTF8EncodingFormat, encoder.Encoding)
	}

	if encoder.encoder == nil {
		t.Error("encoder.encoder should not be nil")
	}
}

func TestTextEncoder_Encode_BasicASCII(t *testing.T) {
	encoder := NewTextEncoder()

	testCases := []struct {
		input    string
		expected []byte
		desc     string
	}{
		{"", []byte{}, "empty string"},
		{"A", []byte{0x41}, "single ASCII character"},
		{"Hello", []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}, "ASCII string"},
		{"Hello World", []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x57, 0x6f, 0x72, 0x6c, 0x64}, "ASCII with space"},
		{"123", []byte{0x31, 0x32, 0x33}, "numeric string"},
		{"!@#$%", []byte{0x21, 0x40, 0x23, 0x24, 0x25}, "special characters"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := encoder.Encode(tc.input)
			if err != nil {
				t.Errorf("Encode(%q) returned error: %v", tc.input, err)
				return
			}

			if !bytes.Equal(result, tc.expected) {
				t.Errorf("Encode(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestTextEncoder_Encode_UTF8(t *testing.T) {
	encoder := NewTextEncoder()

	testCases := []struct {
		input    string
		expected []byte
		desc     string
	}{
		{"café", []byte{0x63, 0x61, 0x66, 0xc3, 0xa9}, "Latin with accent"},
		{"水", []byte{0xe6, 0xb0, 0xb4}, "CJK character"},
		{"Ω", []byte{0xce, 0xa9}, "Greek character"},
		{"€", []byte{0xe2, 0x82, 0xac}, "Euro symbol"},
		{"🌟", []byte{0xf0, 0x9f, 0x8c, 0x9f}, "Emoji"},
		{"α β γ", []byte{0xce, 0xb1, 0x20, 0xce, 0xb2, 0x20, 0xce, 0xb3}, "Greek letters with spaces"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := encoder.Encode(tc.input)
			if err != nil {
				t.Errorf("Encode(%q) returned error: %v", tc.input, err)
				return
			}

			if !bytes.Equal(result, tc.expected) {
				t.Errorf("Encode(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestTextEncoder_Encode_SurrogatePairs(t *testing.T) {
	encoder := NewTextEncoder()

	// Musical symbol G clef (U+1D11E)
	input := "\U0001D11E"
	expected := []byte{0xf0, 0x9d, 0x84, 0x9e}

	result, err := encoder.Encode(input)
	if err != nil {
		t.Errorf("Encode(%q) returned error: %v", input, err)
		return
	}

	if !bytes.Equal(result, expected) {
		t.Errorf("Encode(%q) = %v, expected %v", input, result, expected)
	}
}

func TestTextEncoder_Encode_MixedContent(t *testing.T) {
	encoder := NewTextEncoder()

	testCases := []struct {
		input string
		desc  string
	}{
		{"Hello 世界", "ASCII + CJK"},
		{"Café München", "Latin + German umlaut"},
		{"Price: €100", "ASCII + Euro symbol"},
		{"Test 🚀 rocket", "ASCII + emoji"},
		{"αβγ ABC 123", "Greek + ASCII + numbers"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := encoder.Encode(tc.input)
			if err != nil {
				t.Errorf("Encode(%q) returned error: %v", tc.input, err)
				return
			}

			if len(result) == 0 {
				t.Errorf("Encode(%q) returned empty result", tc.input)
			}

			// For mixed content, byte length should typically be >= string length
			if len(result) < len(tc.input) {
				t.Errorf("Encode(%q) result length %d is less than input length %d",
					tc.input, len(result), len(tc.input))
			}
		})
	}
}

func TestTextEncoder_Encode_LargeString(t *testing.T) {
	encoder := NewTextEncoder()

	// Create a large string with repeated content
	largeString := ""
	for i := 0; i < 1000; i++ {
		largeString += "Hello 世界! "
	}

	result, err := encoder.Encode(largeString)
	if err != nil {
		t.Errorf("Encode(large string) returned error: %v", err)
		return
	}

	if len(result) == 0 {
		t.Error("Encode(large string) returned empty result")
	}

	// Verify the result is reasonable (should be larger than input due to UTF-8 encoding)
	if len(result) < len(largeString) {
		t.Errorf("Encoded result length %d is less than input length %d",
			len(result), len(largeString))
	}
}

func TestTextEncoder_Encode_NilEncoder(t *testing.T) {
	encoder := &TextEncoder{
		Encoding: UTF8EncodingFormat,
		encoder:  nil, // Explicitly set to nil
	}

	_, err := encoder.Encode("test")
	if err == nil {
		t.Error("Expected error when encoder is nil, got nil")
	}

	expectedErr := "encoding not set"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestTextEncoder_Encode_EdgeCases(t *testing.T) {
	encoder := NewTextEncoder()

	testCases := []struct {
		input string
		desc  string
	}{
		{"\x00", "null character"},
		{"\t\n\r", "whitespace characters"},
		{"\u0001\u0002\u0003", "control characters"},
		{"\ufeff", "BOM character"},
		{string([]rune{0x10000}), "4-byte UTF-8 character"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := encoder.Encode(tc.input)
			if err != nil {
				t.Errorf("Encode(%q) returned error: %v", tc.input, err)
				return
			}

			if len(result) == 0 && len(tc.input) > 0 {
				t.Errorf("Encode(%q) returned empty result for non-empty input", tc.input)
			}
		})
	}
}

func BenchmarkTextEncoder_Encode_ASCII(b *testing.B) {
	encoder := NewTextEncoder()
	text := "Hello World"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encoder.Encode(text)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTextEncoder_Encode_UTF8(b *testing.B) {
	encoder := NewTextEncoder()
	text := "Hello 世界! 🌟"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encoder.Encode(text)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTextEncoder_Encode_Large(b *testing.B) {
	encoder := NewTextEncoder()
	largeText := ""
	for i := 0; i < 100; i++ {
		largeText += "Hello 世界! This is a test string with mixed content. 🚀\n"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encoder.Encode(largeText)
		if err != nil {
			b.Fatal(err)
		}
	}
}
