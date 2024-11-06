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

	// Handle errors
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
