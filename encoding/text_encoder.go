package encoding

import (
	"errors"
	"fmt"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

// TextEncoder represents an encoder that will generate a byte stream
// with UTF-8 encoding.
type TextEncoder struct {
	// Encoding always holds the `utf-8` value.
	// FIXME: this should be TextEncoder.prototype.encoding instead
	Encoding EncodingName

	encoder encoding.Encoding
}

// Encodee takes a string as input and returns an encoded byte stream.
func (te *TextEncoder) Encode(text string) ([]byte, error) {
	if te.encoder == nil {
		return nil, errors.New("encoding not set")
	}

	enc := te.encoder.NewEncoder()
	encoded, err := enc.Bytes([]byte(text))
	if err != nil {
		return nil, NewError(TypeError, "unable to encode text; reason: "+err.Error())
	}

	return encoded, nil
}

func newTextEncoder(label EncodingName) (*TextEncoder, error) {
	var encoder encoding.Encoding
	switch label {
	case Windows1252EncodingFormat:
		encoder = charmap.Windows1252
	case UTF8EncodingFormat:
		encoder = unicode.UTF8
	case UTF16LEEncodingFormat:
		encoder = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	case UTF16BEEncodingFormat:
		encoder = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	default:
		return nil, NewError(RangeError, fmt.Sprintf("unsupported encoding: %s", label))
	}

	te := &TextEncoder{
		encoder:  encoder,
		Encoding: label,
	}

	return te, nil
}
