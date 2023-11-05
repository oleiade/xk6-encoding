package encoding

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// TextDecoder represents a decoder for a specific text encoding, such
// as UTF-8, UTF-16, ISO-8859-2, etc.
//
// A decoder takes a stream of bytes as input and emits a stream of code points.
type TextDecoder struct {
	// Encoding holds the name of the decoder which is a string describing
	// the method the `TextDecoder` will use.
	Encoding EncodingName

	// Fatal holds a boolean indicating whether the error mode is fatal.
	Fatal bool

	// IgnoreBOM holds a boolean indicating whether the byte order mark is ignored.
	IgnoreBOM bool

	decoder   encoding.Encoding
	transform transform.Transformer

	rt *goja.Runtime
}

// Decode takes a byte stream as input and returns a string.
func (td *TextDecoder) Decode(buffer []byte, options decodeOptions) (string, error) {
	if td.decoder == nil {
		return "", errors.New("encoding not set")
	}

	var decoded string
	if options.Stream {
		if td.transform == nil {
			td.transform = td.decoder.NewDecoder()
		}

		out, _, err := transform.String(td.transform, string(buffer))
		if err != nil {
			return "", NewError(TypeError, "unable to decode text; reason: "+err.Error())
		}

		decoded = out
	} else {
		decoder := td.decoder.NewDecoder()
		out, err := decoder.String(string(buffer))
		if err != nil {
			return "", NewError(TypeError, "unable to decode text; reason: "+err.Error())
		}
		decoded = out
		td.transform = nil
	}

	return decoded, nil
}

type decodeOptions struct {
	// A boolean flag indicating whether additional data
	// will follow in subsequent calls to decode().
	//
	// Set to true if processing the data in chunks, and
	// false for the final chunk or if the data is not chunked.
	Stream bool `js:"stream"`
}

// NewTextDecoder returns a new TextDecoder object instance that will
// generate a string from a byte stream with a specific encoding.
func NewTextDecoder(rt *goja.Runtime, label string, options textDecoderOptions) (*TextDecoder, error) {
	// Pick the encoding BOM policy accordingly
	bomPolicy := unicode.IgnoreBOM
	if !options.IgnoreBOM {
		bomPolicy = unicode.UseBOM
	}

	var decoder encoding.Encoding
	switch strings.TrimSpace(strings.ToLower(label)) {
	case "",
		"unicode-1-1-utf-8",
		"unicode11utf8",
		"unicode20utf8",
		"utf-8",
		"utf8",
		"x-unicode20utf8":
		label = UTF8EncodingFormat
		decoder = unicode.UTF8
	case UTF16LEEncodingFormat:
		decoder = unicode.UTF16(unicode.LittleEndian, bomPolicy)
	case UTF16BEEncodingFormat:
		decoder = unicode.UTF16(unicode.BigEndian, bomPolicy)
	default:
		return nil, NewError(RangeError, fmt.Sprintf("unsupported encoding: %s", label))
	}

	td := &TextDecoder{
		Encoding:  label,
		IgnoreBOM: options.IgnoreBOM,
		Fatal:     options.Fatal,

		decoder: decoder,
		rt:      rt,
	}

	return td, nil
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

type textDecoderOptions struct {
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
