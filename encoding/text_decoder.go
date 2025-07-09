package encoding

import (
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/sobek"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// TextDecoder represents a decoder for a specific text encoding, such
// as UTF-8, UTF-16, ISO-8859-2, etc.
//
// A decoder takes a stream of bytes as input and emits a stream of code points.
type TextDecoder struct {
	TextDecoderCommon

	decoder   encoding.Encoding
	transform transform.Transformer

	rt *sobek.Runtime

	buffer []byte
}

// TextDecoderOptions represents the options that can be passed to the
// `TextDecoder` constructor.
type TextDecoderOptions struct {
	// Fatal holds a boolean value indicating if
	// the `TextDecoder.decode()`` method must throw
	// a `TypeError` when decoding invalid data.
	//
	// It defaults to `false`, which means that the
	// decoder will substitute malformed data with a
	// replacement character.
	Fatal bool `js:"fatal"`

	// IgnoreBOM holds a boolean value indicating
	// whether the byte order mark is ignored.
	IgnoreBOM bool `js:"ignoreBOM"`
}

// TextDecoderCommon represents the common subset of the TextDecoder interface
// that is shared between the TextDecoder and TextDecoderStream interfaces.
type TextDecoderCommon struct {
	// Encoding holds the name of the decoder which is a string describing
	// the method the `TextDecoder` will use.
	Encoding EncodingName `js:"encoding"`

	// Fatal holds a boolean value indicating if
	// the `TextDecoder.decode()`` method must throw
	// a `TypeError` when decoding invalid data.
	//
	// It defaults to `false`, which means that the
	// decoder will substitute malformed data with a
	// replacement character.
	Fatal bool `js:"fatal"`

	// IgnoreBOM holds a boolean value indicating
	// whether the byte order mark is ignored.
	IgnoreBOM bool `js:"ignoreBOM"`

	errorMode ErrorMode
}

// NewTextDecoder returns a new TextDecoder object instance that will
// generate a string from a byte stream with a specific encoding.
func NewTextDecoder(rt *sobek.Runtime, label string, options TextDecoderOptions) (*TextDecoder, error) {
	// Pick the encoding BOM policy accordingly
	bomPolicy := unicode.IgnoreBOM
	if !options.IgnoreBOM {
		bomPolicy = unicode.UseBOM
	}

	// 1.
	var enc EncodingName
	var decoder encoding.Encoding
	switch strings.TrimSpace(strings.ToLower(label)) {
	case "",
		"unicode-1-1-utf-8",
		"unicode11utf8",
		"unicode20utf8",
		"utf-8",
		"utf8",
		"x-unicode20utf8":
		enc = UTF8EncodingFormat
		decoder = unicode.UTF8
	case "csunicode",
		"iso-10646-ucs-2",
		"ucs-2",
		"unicode",
		"unicodefeff",
		"utf-16",
		"utf-16le":
		enc = UTF16LEEncodingFormat
		decoder = unicode.UTF16(unicode.LittleEndian, bomPolicy)
	case "unicodefffe", "utf-16be":
		enc = UTF16BEEncodingFormat
		decoder = unicode.UTF16(unicode.BigEndian, bomPolicy)
	default:
		// 2.
		return nil, NewError(RangeError, fmt.Sprintf("unsupported enc: %s", label))
	}

	// 3.
	td := &TextDecoder{
		rt: rt,
	}
	td.Encoding = enc
	td.decoder = decoder

	// 4.
	if options.Fatal {
		td.Fatal = true
		td.errorMode = FatalErrorMode
	}

	// 5.
	td.IgnoreBOM = options.IgnoreBOM

	return td, nil
}

// Decode takes a byte stream as input and returns a string.
func (td *TextDecoder) Decode(buffer []byte, options TextDecodeOptions) (string, error) {
	if td.decoder == nil {
		return "", errors.New("encoding not set")
	}

	// Set doNotFlush based on the stream option
	doNotFlush := options.Stream

	// Create the transformer if it's not already created
	if td.transform == nil {
		var transformer transform.Transformer
		decoder := td.decoder.NewDecoder()

		// Configure decoder for BOM handling
		if !td.IgnoreBOM {
			transformer = unicode.BOMOverride(decoder)
		} else {
			transformer = decoder
		}
		td.transform = transformer
	}

	// Store the previous buffer state to detect incomplete sequences
	var prevIncompleteBytes []byte
	if len(td.buffer) > 0 {
		prevIncompleteBytes = make([]byte, len(td.buffer))
		copy(prevIncompleteBytes, td.buffer)
	}

	// Append the new buffer to the internal buffer
	if len(buffer) > 0 {
		td.buffer = append(td.buffer, buffer...)
	}

	// Prepare the dest buffer
	dest := make([]byte, len(td.buffer)*4) // Allocate enough space
	src := td.buffer
	atEOF := !doNotFlush

	destPos, srcPos, err := td.transform.Transform(dest, src, atEOF)

	// Keep any remaining src bytes in td.buffer
	td.buffer = td.buffer[srcPos:]

	// Handle the case where we have incomplete UTF-8 sequence in streaming mode
	if err != nil && errors.Is(err, transform.ErrShortSrc) && doNotFlush {
		// We have an incomplete UTF-8 sequence in streaming mode
		// Case 1: We have previous incomplete bytes and new bytes
		if len(prevIncompleteBytes) > 0 && len(buffer) > 0 {
			// We had previous incomplete bytes and got new bytes
			// Check if the new bytes can complete the sequence
			if !canCompleteUTF8Sequence(prevIncompleteBytes, buffer) {
				// The new bytes can't complete the previous sequence
				// Force emission of replacement characters for the incomplete sequence

				destPos2, _, err2 := td.transform.Transform(dest[destPos:], prevIncompleteBytes, true)
				if err2 == nil || errors.Is(err2, transform.ErrShortDst) {
					destPos += destPos2

					// Reset transformer and process the new bytes
					td.transform.Reset()

					// Now process the new bytes
					if len(buffer) > 0 {
						remainingDest := dest[destPos:]
						// Process new bytes with atEOF=false for streaming, but retry with atEOF=true if we get ErrShortSrc
						destPos3, srcPos3, err3 := td.transform.Transform(remainingDest, buffer, atEOF)

						// If we get ErrShortSrc in streaming mode, try again with atEOF=true to process complete characters
						if err3 != nil && errors.Is(err3, transform.ErrShortSrc) && doNotFlush {
							// Try processing complete characters with atEOF=true
							destPos3, srcPos3, err3 = td.transform.Transform(remainingDest, buffer, true)
						}

						if err3 == nil || errors.Is(err3, transform.ErrShortSrc) {
							destPos += destPos3
							td.buffer = buffer[srcPos3:]
						}
					} else {
						td.buffer = nil
					}
					err = nil // Clear the error since we handled it
				}
			}
		} else if len(prevIncompleteBytes) == 0 && len(buffer) > 0 {
			// Case 2: Single buffer contains incomplete sequence followed by incompatible bytes
			// Find the incomplete UTF-8 sequence at the beginning and check if remaining bytes can complete it
			if len(td.buffer) > 0 {
				// Try to identify the incomplete sequence
				if td.buffer[0]&0xF8 == 0xF0 && len(td.buffer) >= 1 {
					// 4-byte sequence starting with 0xF0
					expectedLength := 4
					if len(td.buffer) < expectedLength {
						// We have an incomplete 4-byte sequence
						incompleteLen := 1 // Start with just the first byte
						for incompleteLen < len(td.buffer) && incompleteLen < expectedLength {
							if td.buffer[incompleteLen]&0xC0 == 0x80 {
								incompleteLen++
							} else {
								break
							}
						}

						// Check if remaining bytes can complete the sequence
						remainingBytes := td.buffer[incompleteLen:]
						if len(remainingBytes) > 0 && !canCompleteUTF8Sequence(td.buffer[:incompleteLen], remainingBytes) {
							// Emit replacement for incomplete sequence
							destPos2, _, err2 := td.transform.Transform(dest[destPos:], td.buffer[:incompleteLen], true)
							if err2 == nil || errors.Is(err2, transform.ErrShortDst) {
								destPos += destPos2

								// Reset transformer and process remaining bytes
								td.transform.Reset()

								remainingDest := dest[destPos:]

								// Process remaining bytes in streaming mode
								if doNotFlush {
									// In streaming mode, try to process complete characters first with atEOF=false
									destPos3, srcPos3, err3 := td.transform.Transform(remainingDest, remainingBytes, false)

									// If we get ErrShortSrc, process complete characters up to the incomplete sequence
									if err3 != nil && errors.Is(err3, transform.ErrShortSrc) {
										// Process complete characters with atEOF=true, but only up to the incomplete sequence
										i := 0
										for i < len(remainingBytes) {
											if remainingBytes[i]&0x80 == 0 {
												// ASCII character, can be processed
												i++
											} else {
												// Start of potentially incomplete UTF-8 sequence
												break
											}
										}

										if i > 0 {
											// Process the complete ASCII characters
											completeBytes := remainingBytes[:i]
											destPos3, srcPos3, err3 = td.transform.Transform(remainingDest, completeBytes, true)
											if err3 == nil {
												destPos += destPos3
												td.buffer = remainingBytes[i:]
											}
										} else {
											// No complete characters, just buffer everything
											td.buffer = remainingBytes
										}
									} else {
										// No error or different error, use the result
										destPos += destPos3
										td.buffer = remainingBytes[srcPos3:]
									}
								} else {
									// In non-streaming mode, process everything
									destPos3, srcPos3, err3 := td.transform.Transform(remainingDest, remainingBytes, true)
									if err3 == nil || errors.Is(err3, transform.ErrShortSrc) {
										destPos += destPos3
										td.buffer = remainingBytes[srcPos3:]
									}
								}
								err = nil // Clear the error since we handled it
							}
						}
					}
				}
			}
		}
	}

	// Handle other errors
	if err != nil && !errors.Is(err, transform.ErrShortSrc) && !errors.Is(err, transform.ErrShortDst) {
		if td.Fatal {
			return "", NewError(TypeError, "unable to decode text; reason: "+err.Error())
		}
		// In non-fatal mode, continue with replacement characters
		// The golang.org/x/text/transform package should handle this automatically
	}

	decoded := string(dest[:destPos])

	// In fatal mode, check if any replacement characters were inserted
	if td.Fatal && decoded != "" {
		// Check for UTF-8 replacement character (U+FFFD)
		for _, r := range decoded {
			if r == '\uFFFD' {
				return "", NewError(TypeError, "invalid byte sequence")
			}
		}
	}

	// Reset the transformer and buffer when not streaming
	if !doNotFlush {
		td.transform.Reset()
		td.transform = nil
		td.buffer = nil
	}

	return decoded, nil
}

// canCompleteUTF8Sequence checks if the new bytes can potentially complete
// the incomplete UTF-8 sequence in the buffer
func canCompleteUTF8Sequence(incompleteBuffer, newBytes []byte) bool {
	return canCompleteUTF8SequenceImpl(incompleteBuffer, newBytes)
}

// CanCompleteUTF8SequenceDebug is exported for testing
func CanCompleteUTF8SequenceDebug(incompleteBuffer, newBytes []byte) bool {
	return canCompleteUTF8SequenceImpl(incompleteBuffer, newBytes)
}

func canCompleteUTF8SequenceImpl(incompleteBuffer, newBytes []byte) bool {
	if len(incompleteBuffer) == 0 || len(newBytes) == 0 {
		return false
	}

	// Check the first byte to determine expected sequence length
	firstByte := incompleteBuffer[0]
	var expectedLength int

	if firstByte&0x80 == 0 {
		// ASCII character (0xxxxxxx) - already complete
		return false
	} else if firstByte&0xE0 == 0xC0 {
		// 2-byte sequence (110xxxxx)
		expectedLength = 2
	} else if firstByte&0xF0 == 0xE0 {
		// 3-byte sequence (1110xxxx)
		expectedLength = 3
	} else if firstByte&0xF8 == 0xF0 {
		// 4-byte sequence (11110xxx)
		expectedLength = 4
	} else {
		// Invalid UTF-8 start byte
		return false
	}

	// Check if we have enough bytes to complete the sequence
	totalBytesNeeded := expectedLength - len(incompleteBuffer)
	if len(newBytes) < totalBytesNeeded {
		// Not enough bytes, might be able to complete later
		// Check if all new bytes are valid continuation bytes and follow UTF-8 rules
		for i, b := range newBytes {
			if !isValidUTF8ContinuationByte(incompleteBuffer, i+len(incompleteBuffer), b) {
				return false
			}
		}
		return true // Could potentially complete with more bytes
	}

	// We have enough bytes, check if the first `totalBytesNeeded` bytes
	// are valid continuation bytes and follow UTF-8 rules
	for i := 0; i < totalBytesNeeded; i++ {
		if !isValidUTF8ContinuationByte(incompleteBuffer, i+len(incompleteBuffer), newBytes[i]) {
			return false
		}
	}

	return true
}

// isValidUTF8ContinuationByte checks if a byte is a valid continuation byte
// for a UTF-8 sequence at the given position
func isValidUTF8ContinuationByte(incompleteBuffer []byte, position int, b byte) bool {
	// All continuation bytes must have the pattern 10xxxxxx
	if b&0xC0 != 0x80 {
		return false
	}

	// Additional validation based on the first byte and position
	if len(incompleteBuffer) == 0 {
		return false
	}

	firstByte := incompleteBuffer[0]

	// For 4-byte sequences starting with 0xF0, the second byte must be 0x90-0xBF
	if firstByte == 0xF0 && position == 1 {
		return b >= 0x90 && b <= 0xBF
	}

	// For 4-byte sequences starting with 0xF4, the second byte must be 0x80-0x8F
	if firstByte == 0xF4 && position == 1 {
		return b >= 0x80 && b <= 0x8F
	}

	// For 3-byte sequences starting with 0xE0, the second byte must be 0xA0-0xBF
	if firstByte == 0xE0 && position == 1 {
		return b >= 0xA0 && b <= 0xBF
	}

	// For 3-byte sequences starting with 0xED, the second byte must be 0x80-0x9F
	if firstByte == 0xED && position == 1 {
		return b >= 0x80 && b <= 0x9F
	}

	// For 2-byte sequences starting with 0xC0 or 0xC1, they are invalid
	if (firstByte == 0xC0 || firstByte == 0xC1) && position == 1 {
		return false
	}

	// For other cases, any continuation byte is valid
	return true
}

// GetBufferForDebug returns the internal buffer for debugging purposes
func (td *TextDecoder) GetBufferForDebug() []byte {
	return td.buffer
}

// TextDecodeOptions represents the options that can be passed to the
// TextDecoder.decode() method.
type TextDecodeOptions struct {
	// A boolean flag indicating whether additional data
	// will follow in subsequent calls to decode().
	//
	// Set to true if processing the data in chunks, and
	// false for the final chunk or if the data is not chunked.
	Stream bool `js:"stream"`
}

// EncodingName is a type alias for the name of an encoding.
//
//nolint:revive
type EncodingName = string

const (
	// UTF8EncodingFormat is the encoding format for utf-8
	UTF8EncodingFormat = "utf-8"

	// UTF16LEEncodingFormat is the encoding format for utf-16le
	UTF16LEEncodingFormat = "utf-16le"

	// UTF16BEEncodingFormat is the encoding format for utf-16be
	UTF16BEEncodingFormat = "utf-16be"
)

// ErrorMode is a type alias for the error mode of a TextDecoder.
type ErrorMode = string

const (
	// ReplacementErrorMode is the error mode for replacing
	// invalid characters with the replacement character.
	ReplacementErrorMode ErrorMode = "replacement"

	// FatalErrorMode is the error mode for throwing a
	// TypeError when an invalid character is encountered.
	FatalErrorMode ErrorMode = "fatal"
)
