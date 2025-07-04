package encoding

import (
	"testing"

	"github.com/grafana/sobek"
)

func TestNewTextDecoder_UTF8(t *testing.T) {
	rt := sobek.New()

	testCases := []struct {
		label   string
		options textDecoderOptions
		desc    string
	}{
		{"", textDecoderOptions{}, "empty label defaults to UTF-8"},
		{"utf-8", textDecoderOptions{}, "explicit UTF-8"},
		{"UTF-8", textDecoderOptions{}, "uppercase UTF-8"},
		{"unicode-1-1-utf-8", textDecoderOptions{}, "unicode-1-1-utf-8 label"},
		{"unicode11utf8", textDecoderOptions{}, "unicode11utf8 label"},
		{"unicode20utf8", textDecoderOptions{}, "unicode20utf8 label"},
		{"utf8", textDecoderOptions{}, "utf8 without dash"},
		{"x-unicode20utf8", textDecoderOptions{}, "x-unicode20utf8 label"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			decoder, err := NewTextDecoder(rt, tc.label, tc.options)
			if err != nil {
				t.Errorf("NewTextDecoder(%q) returned error: %v", tc.label, err)
				return
			}

			if decoder == nil {
				t.Fatal("NewTextDecoder() should not return nil")
			}

			if decoder.Encoding != UTF8EncodingFormat {
				t.Errorf("Expected encoding to be %s, got %s", UTF8EncodingFormat, decoder.Encoding)
			}

			if decoder.decoder == nil {
				t.Error("decoder.decoder should not be nil")
			}

			if decoder.Fatal != tc.options.Fatal {
				t.Errorf("Expected Fatal to be %v, got %v", tc.options.Fatal, decoder.Fatal)
			}

			if decoder.IgnoreBOM != tc.options.IgnoreBOM {
				t.Errorf("Expected IgnoreBOM to be %v, got %v", tc.options.IgnoreBOM, decoder.IgnoreBOM)
			}
		})
	}
}

func TestNewTextDecoder_UTF16(t *testing.T) {
	rt := sobek.New()

	testCases := []struct {
		label    string
		expected string
		desc     string
	}{
		{"utf-16le", UTF16LEEncodingFormat, "UTF-16LE"},
		{"utf-16be", UTF16BEEncodingFormat, "UTF-16BE"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			decoder, err := NewTextDecoder(rt, tc.label, textDecoderOptions{})
			if err != nil {
				t.Errorf("NewTextDecoder(%q) returned error: %v", tc.label, err)
				return
			}

			if decoder.Encoding != tc.expected {
				t.Errorf("Expected encoding to be %s, got %s", tc.expected, decoder.Encoding)
			}
		})
	}
}

func TestNewTextDecoder_UnsupportedEncoding(t *testing.T) {
	rt := sobek.New()

	unsupportedEncodings := []string{
		"iso-8859-1",
		"windows-1252",
		"ascii",
		"latin1",
		"unknown-encoding",
	}

	for _, encoding := range unsupportedEncodings {
		t.Run(encoding, func(t *testing.T) {
			_, err := NewTextDecoder(rt, encoding, textDecoderOptions{})
			if err == nil {
				t.Errorf("Expected error for unsupported encoding %q, got nil", encoding)
				return
			}

			if err.Error() != "RangeError: unsupported encoding: "+encoding {
				t.Errorf("Expected RangeError for unsupported encoding, got: %v", err)
			}
		})
	}
}

func TestNewTextDecoder_WithOptions(t *testing.T) {
	rt := sobek.New()

	options := textDecoderOptions{
		Fatal:     true,
		IgnoreBOM: true,
	}

	decoder, err := NewTextDecoder(rt, "utf-8", options)
	if err != nil {
		t.Fatalf("NewTextDecoder() returned error: %v", err)
	}

	if !decoder.Fatal {
		t.Error("Expected Fatal to be true")
	}

	if !decoder.IgnoreBOM {
		t.Error("Expected IgnoreBOM to be true")
	}
}

func TestTextDecoder_Decode_UTF8_BasicASCII(t *testing.T) {
	rt := sobek.New()
	decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{})
	if err != nil {
		t.Fatalf("NewTextDecoder() failed: %v", err)
	}

	testCases := []struct {
		input    []byte
		expected string
		desc     string
	}{
		{[]byte{}, "", "empty input"},
		{[]byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}, "Hello", "ASCII string"},
		{[]byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x57, 0x6f, 0x72, 0x6c, 0x64}, "Hello World", "ASCII with space"},
		{[]byte{0x31, 0x32, 0x33}, "123", "numeric string"},
		{[]byte{0x21, 0x40, 0x23, 0x24, 0x25}, "!@#$%", "special characters"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := decoder.Decode(tc.input, decodeOptions{})
			if err != nil {
				t.Errorf("Decode() returned error: %v", err)
				return
			}

			if result != tc.expected {
				t.Errorf("Decode() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

func TestTextDecoder_Decode_UTF8_Unicode(t *testing.T) {
	rt := sobek.New()
	decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{})
	if err != nil {
		t.Fatalf("NewTextDecoder() failed: %v", err)
	}

	testCases := []struct {
		input    []byte
		expected string
		desc     string
	}{
		{[]byte{0x63, 0x61, 0x66, 0xc3, 0xa9}, "café", "Latin with accent"},
		{[]byte{0xe6, 0xb0, 0xb4}, "水", "CJK character"},
		{[]byte{0xce, 0xa9}, "Ω", "Greek character"},
		{[]byte{0xe2, 0x82, 0xac}, "€", "Euro symbol"},
		{[]byte{0xf0, 0x9f, 0x8c, 0x9f}, "🌟", "Emoji"},
		{[]byte{0xce, 0xb1, 0x20, 0xce, 0xb2, 0x20, 0xce, 0xb3}, "α β γ", "Greek letters with spaces"},
		{[]byte{0xf0, 0x9d, 0x84, 0x9e}, "\U0001D11E", "Musical symbol G clef"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := decoder.Decode(tc.input, decodeOptions{})
			if err != nil {
				t.Errorf("Decode() returned error: %v", err)
				return
			}

			if result != tc.expected {
				t.Errorf("Decode() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

func TestTextDecoder_Decode_UTF8_BOM(t *testing.T) {
	rt := sobek.New()

	testData := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f} // "Hello"
	bomData := []byte{0xef, 0xbb, 0xbf}              // UTF-8 BOM
	expected := "Hello"

	t.Run("BOM respected by default", func(t *testing.T) {
		decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{IgnoreBOM: false})
		if err != nil {
			t.Fatalf("NewTextDecoder() failed: %v", err)
		}

		// Data with BOM
		input := append(bomData, testData...)
		result, err := decoder.Decode(input, decodeOptions{})
		if err != nil {
			t.Errorf("Decode() returned error: %v", err)
			return
		}

		if result != expected {
			t.Errorf("Decode() with BOM = %q, expected %q", result, expected)
		}
	})

	t.Run("BOM ignored when IgnoreBOM is true", func(t *testing.T) {
		decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{IgnoreBOM: true})
		if err != nil {
			t.Fatalf("NewTextDecoder() failed: %v", err)
		}

		// Data with BOM
		input := append(bomData, testData...)
		result, err := decoder.Decode(input, decodeOptions{})
		if err != nil {
			t.Errorf("Decode() returned error: %v", err)
			return
		}

		// Should include BOM character in output when IgnoreBOM is true
		if len(result) <= len(expected) {
			t.Errorf("Expected result to include BOM character, got %q", result)
		}
	})
}

func TestTextDecoder_Decode_Streaming(t *testing.T) {
	rt := sobek.New()
	decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{})
	if err != nil {
		t.Fatalf("NewTextDecoder() failed: %v", err)
	}

	// Test streaming with complete characters in each chunk
	// This tests the streaming functionality without incomplete UTF-8 sequences
	chunk1 := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20} // "Hello "
	chunk2 := []byte{0xe6, 0xb0, 0xb4}                   // "水"
	chunk3 := []byte{0x20, 0x57, 0x6f, 0x72, 0x6c, 0x64} // " World"

	// First chunk with streaming enabled
	result1, err := decoder.Decode(chunk1, decodeOptions{Stream: true})
	if err != nil {
		t.Errorf("Decode(chunk1) returned error: %v", err)
		return
	}

	// Second chunk with streaming enabled
	result2, err := decoder.Decode(chunk2, decodeOptions{Stream: true})
	if err != nil {
		t.Errorf("Decode(chunk2) returned error: %v", err)
		return
	}

	// Third chunk (final chunk)
	result3, err := decoder.Decode(chunk3, decodeOptions{Stream: false})
	if err != nil {
		t.Errorf("Decode(chunk3) returned error: %v", err)
		return
	}

	fullResult := result1 + result2 + result3
	expected := "Hello 水 World"

	if fullResult != expected {
		t.Errorf("Streaming decode result = %q, expected %q", fullResult, expected)
	}
}

func TestTextDecoder_Decode_NilDecoder(t *testing.T) {
	decoder := &TextDecoder{
		Encoding:  UTF8EncodingFormat,
		decoder:   nil, // Explicitly set to nil
		Fatal:     false,
		IgnoreBOM: false,
	}

	_, err := decoder.Decode([]byte("test"), decodeOptions{})
	if err == nil {
		t.Error("Expected error when decoder is nil, got nil")
	}

	expectedErr := "encoding not set"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestTextDecoder_Decode_EdgeCases(t *testing.T) {
	rt := sobek.New()
	decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{})
	if err != nil {
		t.Fatalf("NewTextDecoder() failed: %v", err)
	}

	testCases := []struct {
		input []byte
		desc  string
	}{
		{[]byte{0x00}, "null byte"},
		{[]byte{0x09, 0x0a, 0x0d}, "tab, newline, carriage return"},
		{[]byte{0x01, 0x02, 0x03}, "control characters"},
		{[]byte{0x7f}, "DEL character"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := decoder.Decode(tc.input, decodeOptions{})
			if err != nil {
				t.Errorf("Decode() returned error for %s: %v", tc.desc, err)
				return
			}

			if len(result) == 0 && len(tc.input) > 0 {
				t.Errorf("Decode() returned empty result for non-empty input: %s", tc.desc)
			}
		})
	}
}

func TestTextDecoder_Decode_LargeInput(t *testing.T) {
	rt := sobek.New()
	decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{})
	if err != nil {
		t.Fatalf("NewTextDecoder() failed: %v", err)
	}

	// Create large input with repeated UTF-8 content
	base := []byte("Hello 世界! 🌟\n")
	var largeInput []byte
	for i := 0; i < 1000; i++ {
		largeInput = append(largeInput, base...)
	}

	result, err := decoder.Decode(largeInput, decodeOptions{})
	if err != nil {
		t.Errorf("Decode(large input) returned error: %v", err)
		return
	}

	if len(result) == 0 {
		t.Error("Decode(large input) returned empty result")
	}

	// Result should contain the expected pattern
	expectedPattern := "Hello 世界! 🌟\n"
	if len(result) < len(expectedPattern) {
		t.Errorf("Result too short: got %d characters, expected at least %d",
			len(result), len(expectedPattern))
	}
}

func TestTextDecoder_Decode_WithWhitespaceLabels(t *testing.T) {
	rt := sobek.New()

	whitespace := []string{" ", "\t", "\n", "\f", "\r"}
	baseLabel := "utf-8"

	for _, ws := range whitespace {
		testCases := []struct {
			label string
			desc  string
		}{
			{ws + baseLabel, "leading whitespace"},
			{baseLabel + ws, "trailing whitespace"},
			{ws + baseLabel + ws, "surrounding whitespace"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc+"_"+string([]byte{ws[0]}), func(t *testing.T) {
				decoder, err := NewTextDecoder(rt, tc.label, textDecoderOptions{})
				if err != nil {
					t.Errorf("NewTextDecoder(%q) returned error: %v", tc.label, err)
					return
				}

				if decoder.Encoding != UTF8EncodingFormat {
					t.Errorf("Expected encoding to be %s, got %s", UTF8EncodingFormat, decoder.Encoding)
				}

				// Test that it can actually decode
				testInput := []byte("Hello")
				result, err := decoder.Decode(testInput, decodeOptions{})
				if err != nil {
					t.Errorf("Decode() failed: %v", err)
					return
				}

				if result != "Hello" {
					t.Errorf("Decode() = %q, expected %q", result, "Hello")
				}
			})
		}
	}
}

func BenchmarkTextDecoder_Decode_ASCII(b *testing.B) {
	rt := sobek.New()
	decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{})
	if err != nil {
		b.Fatalf("NewTextDecoder() failed: %v", err)
	}

	input := []byte("Hello World")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decoder.Decode(input, decodeOptions{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTextDecoder_Decode_UTF8(b *testing.B) {
	rt := sobek.New()
	decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{})
	if err != nil {
		b.Fatalf("NewTextDecoder() failed: %v", err)
	}

	input := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0xe6, 0xb0, 0xb4, 0x21, 0x20, 0xf0, 0x9f, 0x8c, 0x9f}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decoder.Decode(input, decodeOptions{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTextDecoder_Decode_Large(b *testing.B) {
	rt := sobek.New()
	decoder, err := NewTextDecoder(rt, "utf-8", textDecoderOptions{})
	if err != nil {
		b.Fatalf("NewTextDecoder() failed: %v", err)
	}

	// Create large input
	base := []byte("Hello 世界! This is a test string with mixed content. 🚀\n")
	var largeInput []byte
	for i := 0; i < 100; i++ {
		largeInput = append(largeInput, base...)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decoder.Decode(largeInput, decodeOptions{})
		if err != nil {
			b.Fatal(err)
		}
	}
}
