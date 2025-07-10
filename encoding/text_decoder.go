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

	// For UTF-8 in streaming mode, detect and handle invalid sequences immediately
	var replacementCount int
	var incompleteSequences []byte
	if td.Encoding == UTF8EncodingFormat && len(td.buffer) > 0 {
		if doNotFlush {
			// In streaming mode, check if previous incomplete sequences can be completed
			if len(prevIncompleteBytes) > 0 {
				if len(buffer) > 0 {
					// Check if the new bytes can complete the previous incomplete sequence
					if !canCompleteUTF8Sequence(prevIncompleteBytes, buffer) {
						// The previous incomplete sequence cannot be completed
						// Emit replacement characters for it and process new bytes separately
						replacementCount = 1 // One replacement for the incomplete sequence


						// Process only the new bytes, not the incomplete sequence
						invalidSeqs, validBytes, incompleteBytes := detectInvalidUTF8Sequences(buffer)
						replacementCount += len(invalidSeqs)
						td.buffer = validBytes
						incompleteSequences = incompleteBytes

					} else {
						// The new bytes might complete the sequence, process normally
						invalidSeqs, validBytes, incompleteBytes := detectInvalidUTF8Sequences(td.buffer)
						replacementCount = len(invalidSeqs)
						td.buffer = validBytes
						incompleteSequences = incompleteBytes
					}
				} else {
					// No new bytes, but we have previous incomplete sequences
					// This is the case for decode(undefined) - we should flush the incomplete sequence
					// But since we're in streaming mode, we need to preserve it for later
					invalidSeqs, validBytes, incompleteBytes := detectInvalidUTF8Sequences(td.buffer)
					replacementCount = len(invalidSeqs)
					td.buffer = validBytes
					incompleteSequences = incompleteBytes
				}
			} else {
				// No previous incomplete sequences - process normally
				invalidSeqs, validBytes, incompleteBytes := detectInvalidUTF8Sequences(td.buffer)
				replacementCount = len(invalidSeqs)
				td.buffer = validBytes
				incompleteSequences = incompleteBytes
			}
		}
		// In flush mode (!doNotFlush), we'll handle incomplete sequences in the flush section
	}

	// Prepare the dest buffer (account for replacement characters)
	// Ensure minimum size for at least one replacement character in case of incomplete sequences
	minSize := len(td.buffer)*4 + replacementCount*3
	if minSize < 12 { // At least space for 4 replacement characters
		minSize = 12
	}
	dest := make([]byte, minSize)
	destPos := 0

	// Add replacement characters for invalid sequences found in streaming mode
	for i := 0; i < replacementCount; i++ {
		copy(dest[destPos:], "\uFFFD") // UTF-8 encoded replacement character
		destPos += 3
	}

	// Transform valid bytes
	src := td.buffer
	atEOF := !doNotFlush

	newDestPos, srcPos, err := td.transform.Transform(dest[destPos:], src, atEOF)
	destPos += newDestPos

	// Keep any remaining src bytes in td.buffer plus incomplete sequences
	if doNotFlush && td.Encoding == UTF8EncodingFormat && len(incompleteSequences) > 0 {
		// Keep any unprocessed bytes plus the incomplete sequences
		remainingProcessed := td.buffer[srcPos:]
		td.buffer = append(remainingProcessed, incompleteSequences...)
	} else {
		td.buffer = td.buffer[srcPos:]
	}

	// Handle incomplete sequences in streaming mode
	if err != nil && errors.Is(err, transform.ErrShortSrc) && doNotFlush {
		// For UTF-16 encodings, we need to handle incomplete 2-byte sequences
		if td.Encoding == UTF16LEEncodingFormat || td.Encoding == UTF16BEEncodingFormat {
			// UTF-16 characters are 2 bytes each (ignoring surrogates for now)
			// If we have odd number of bytes, keep the last byte buffered
			if len(td.buffer) > 0 {
				// This is expected for UTF-16 streaming with incomplete sequences
				// Just keep the incomplete bytes in the buffer
				err = nil
			}
		} else if len(incompleteSequences) > 0 {
			// For UTF-8, if we deliberately separated incomplete sequences, don't process them now
			// Just clear the error and keep the incomplete sequences in the buffer
			err = nil
		} else {
			// For UTF-8, ErrShortSrc in streaming mode means we have incomplete sequences
			// Try to process complete characters by forcing atEOF=true for malformed sequences
			if len(td.buffer) > 0 {
				// Try to process the buffer with atEOF=true to get replacement characters
				tempDest := make([]byte, len(td.buffer)*4)
				tempDestPos, tempSrcPos, tempErr := td.transform.Transform(tempDest, td.buffer, true)

				// If we successfully processed some characters, use that result
				if tempErr == nil || tempSrcPos > 0 {
					copy(dest[destPos:], tempDest[:tempDestPos])
					destPos += tempDestPos
					td.buffer = td.buffer[tempSrcPos:]
					err = nil
				} else {
					// Just clear the error - keep bytes buffered
					err = nil
				}
			} else {
				err = nil
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

	// Reset the transformer and buffer when not streaming
	if !doNotFlush {
		// If we're not streaming (flushing), any remaining bytes should be treated as incomplete sequences
		if len(td.buffer) > 0 {
			// For UTF-8, directly count and handle incomplete sequences
			if td.Encoding == UTF8EncodingFormat {
				flushReplacementCount := 0
				i := 0

				for i < len(td.buffer) {
					b := td.buffer[i]
					if b < 0x80 {
						// ASCII character - should have been processed already, but handle it
						i++
					} else if b >= 0xC0 && b <= 0xDF {
						// 2-byte sequence starter - incomplete since we're in flush mode
						flushReplacementCount++
						i++
					} else if b >= 0xE0 && b <= 0xEF {
						// 3-byte sequence starter - incomplete since we're in flush mode
						flushReplacementCount++
						i++
					} else if b >= 0xF0 && b <= 0xF7 {
						// 4-byte sequence starter - incomplete since we're in flush mode
						flushReplacementCount++
						i++
					} else {
						// Invalid byte (continuation byte without starter, or invalid range)
						flushReplacementCount++
						i++
					}
				}

				// Add replacement characters for the incomplete sequences
				if flushReplacementCount > 0 {
					// Make sure we have enough space in dest
					needed := flushReplacementCount * 3 // UTF-8 replacement char is 3 bytes
					if destPos+needed > len(dest) {
						// Reallocate dest if needed
						newDest := make([]byte, destPos+needed)
						copy(newDest, dest[:destPos])
						dest = newDest
					}

					// Add replacement characters for incomplete sequences
					for j := 0; j < flushReplacementCount; j++ {
						// Make sure we have enough space
						if destPos+3 > len(dest) {
							newDest := make([]byte, destPos+3)
							copy(newDest, dest[:destPos])
							dest = newDest
						}
						copy(dest[destPos:], "\uFFFD")
						destPos += 3
					}
				}
			} else {
				// For UTF-16, try to process remaining bytes with atEOF=true first
				tempDest := make([]byte, len(td.buffer)*4)
				tempDestPos, _, tempErr := td.transform.Transform(tempDest, td.buffer, true)

				// If we got some output, append it
				if tempDestPos > 0 {
					// Make sure we have enough space in dest
					if destPos+tempDestPos > len(dest) {
						// Reallocate dest if needed
						newDest := make([]byte, destPos+tempDestPos)
						copy(newDest, dest[:destPos])
						dest = newDest
					}
					copy(dest[destPos:], tempDest[:tempDestPos])
					destPos += tempDestPos
				}

				// If there was an error or we didn't process all bytes, add replacement characters
				if tempErr != nil || tempDestPos == 0 {
					// Any remaining bytes are incomplete sequences
					// Each incomplete sequence gets one replacement character
					if len(td.buffer) > 0 {
						// Make sure we have enough space in dest
						needed := 3 // UTF-8 replacement char is 3 bytes
						if destPos+needed > len(dest) {
							// Reallocate dest if needed
							newDest := make([]byte, destPos+needed)
							copy(newDest, dest[:destPos])
							dest = newDest
						}

						// Add one replacement character for the incomplete UTF-16 sequence
						copy(dest[destPos:], "\uFFFD")
						destPos += 3
					}
				}
			}
		}

		td.transform.Reset()
		td.transform = nil
		td.buffer = nil
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

// detectInvalidUTF8Sequences detects invalid UTF-8 patterns that should immediately emit replacement characters
// Returns invalid sequences, valid bytes, and incomplete sequences separately
func detectInvalidUTF8Sequences(buffer []byte) (invalid [][]byte, valid []byte, incomplete []byte) {
	var invalidSeqs [][]byte
	validBytes := make([]byte, 0, len(buffer))
	var incompleteBytes []byte

	i := 0
	for i < len(buffer) {
		b := buffer[i]

		// ASCII bytes are always valid
		if b < 0x80 {
			validBytes = append(validBytes, b)
			i++
			continue
		}

		// Check for invalid UTF-8 patterns
		if b >= 0xC0 && b <= 0xDF {
			// 2-byte sequence starter
			if i+1 >= len(buffer) {
				// Incomplete sequence - save it separately, don't pass to transform
				incompleteBytes = append(incompleteBytes, buffer[i:]...)
				break
			} else if buffer[i+1]&0xC0 != 0x80 {
				// Invalid continuation byte - emit replacement for starter byte
				invalidSeqs = append(invalidSeqs, []byte{b})
				i++
				continue
			} else {
				// Valid 2-byte sequence
				validBytes = append(validBytes, buffer[i:i+2]...)
				i += 2
				continue
			}
		} else if b >= 0xE0 && b <= 0xEF {
			// 3-byte sequence starter
			if i+1 >= len(buffer) {
				// Incomplete sequence - save it separately
				incompleteBytes = append(incompleteBytes, buffer[i:]...)
				break
			} else if buffer[i+1]&0xC0 != 0x80 {
				// Invalid continuation byte - emit replacement for starter byte
				invalidSeqs = append(invalidSeqs, []byte{b})
				i++
				continue
			} else if i+2 >= len(buffer) {
				// Incomplete sequence - save it separately
				incompleteBytes = append(incompleteBytes, buffer[i:]...)
				break
			} else if buffer[i+2]&0xC0 != 0x80 {
				// Invalid second continuation byte - emit replacement for incomplete sequence
				invalidSeqs = append(invalidSeqs, buffer[i:i+2])
				i += 2
				continue
			} else {
				// Check for overlong encodings and surrogates
				if (b == 0xE0 && buffer[i+1] < 0xA0) ||
					(b == 0xED && buffer[i+1] >= 0xA0) {
					// Overlong encoding or surrogate - emit replacement
					invalidSeqs = append(invalidSeqs, buffer[i:i+3])
					i += 3
					continue
				}
				// Valid 3-byte sequence
				validBytes = append(validBytes, buffer[i:i+3]...)
				i += 3
				continue
			}
		} else if b >= 0xF0 && b <= 0xF7 {
			// 4-byte sequence starter
			if i+1 >= len(buffer) {
				// Incomplete sequence - save it separately
				incompleteBytes = append(incompleteBytes, buffer[i:]...)
				break
			} else if buffer[i+1]&0xC0 != 0x80 {
				// Invalid continuation byte - emit replacement for starter byte
				invalidSeqs = append(invalidSeqs, []byte{b})
				i++
				continue
			} else if i+2 >= len(buffer) {
				// Incomplete sequence - save it separately
				incompleteBytes = append(incompleteBytes, buffer[i:]...)
				break
			} else if buffer[i+2]&0xC0 != 0x80 {
				// Invalid second continuation byte - emit replacement for incomplete sequence
				invalidSeqs = append(invalidSeqs, buffer[i:i+2])
				i += 2
				continue
			} else if i+3 >= len(buffer) {
				// Incomplete sequence - save it separately
				incompleteBytes = append(incompleteBytes, buffer[i:]...)
				break
			} else if buffer[i+3]&0xC0 != 0x80 {
				// Invalid third continuation byte - emit replacement for incomplete sequence
				invalidSeqs = append(invalidSeqs, buffer[i:i+3])
				i += 3
				continue
			} else {
				// Check for overlong encodings and out-of-range values
				if (b == 0xF0 && buffer[i+1] < 0x90) ||
					(b == 0xF4 && buffer[i+1] >= 0x90) ||
					b >= 0xF5 {
					// Overlong encoding or out of range - emit replacement
					invalidSeqs = append(invalidSeqs, buffer[i:i+4])
					i += 4
					continue
				}
				// Valid 4-byte sequence
				validBytes = append(validBytes, buffer[i:i+4]...)
				i += 4
				continue
			}
		} else {
			// Invalid byte (0x80-0xBF standalone, 0xF8-0xFF)
			invalidSeqs = append(invalidSeqs, []byte{b})
			i++
			continue
		}
	}

	return invalidSeqs, validBytes, incompleteBytes
}

// separateIncompleteUTF8Sequences separates complete sequences from incomplete ones
// Returns the complete sequences and the length of incomplete sequences at the end
func separateIncompleteUTF8Sequences(buffer []byte) ([]byte, int) {
	if len(buffer) == 0 {
		return buffer, 0
	}

	// Simple case: check if the last byte is a multi-byte sequence starter
	lastByte := buffer[len(buffer)-1]
	if lastByte >= 0xC0 && lastByte <= 0xDF {
		// 2-byte sequence starter at the end - incomplete
		return buffer[:len(buffer)-1], 1
	} else if lastByte >= 0xE0 && lastByte <= 0xEF {
		// 3-byte sequence starter at the end - incomplete
		return buffer[:len(buffer)-1], 1
	} else if lastByte >= 0xF0 && lastByte <= 0xF7 {
		// 4-byte sequence starter at the end - incomplete
		return buffer[:len(buffer)-1], 1
	}

	// For more complex cases with partial sequences, use a more thorough approach
	// This handles cases where we have continuation bytes at the end
	for i := len(buffer) - 1; i >= 0; i-- {
		b := buffer[i]

		if b >= 0xC0 && b <= 0xDF {
			// 2-byte sequence starter
			expectedLen := 2
			actualLen := len(buffer) - i
			if actualLen < expectedLen {
				return buffer[:i], actualLen
			}
		} else if b >= 0xE0 && b <= 0xEF {
			// 3-byte sequence starter
			expectedLen := 3
			actualLen := len(buffer) - i
			if actualLen < expectedLen {
				return buffer[:i], actualLen
			}
		} else if b >= 0xF0 && b <= 0xF7 {
			// 4-byte sequence starter
			expectedLen := 4
			actualLen := len(buffer) - i
			if actualLen < expectedLen {
				return buffer[:i], actualLen
			}
		}
	}

	// No incomplete sequence found
	return buffer, 0
}

// isIncompleteUTF8Sequence checks if the given bytes form an incomplete UTF-8 sequence
func isIncompleteUTF8Sequence(bytes []byte) bool {
	if len(bytes) == 0 {
		return false
	}

	b := bytes[0]

	// Check if it's a multi-byte sequence starter
	if b >= 0xC0 && b <= 0xDF {
		// 2-byte sequence starter
		return len(bytes) < 2
	} else if b >= 0xE0 && b <= 0xEF {
		// 3-byte sequence starter
		return len(bytes) < 3
	} else if b >= 0xF0 && b <= 0xF7 {
		// 4-byte sequence starter
		return len(bytes) < 4
	}

	// Single byte or invalid starter
	return false
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
