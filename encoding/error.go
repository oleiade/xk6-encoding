package encoding

import "fmt"

// ErrorName is a type alias for the name of an encoding error.
//
// Note that it is a type alias, and not a binding, so that it
// is not interpreted as an object by goja.
type ErrorName = string

const (
	// RangeError is thrown if the value of label is unknown, or
	// is one of the values leading to a 'replacement' decoding
	// algorithm ("iso-2022-cn" or "iso-2022-cn-ext").
	RangeError ErrorName = "RangeError"

	// TypeError is thrown if the value if the Decoder fatal option
	// is set and the input data cannot be decoded.
	TypeError ErrorName = "TypeError"
)

// Error represents an encoding error.
type Error struct {
	// Name contains one of the strings associated with an error name.
	Name ErrorName `json:"name"`

	// Message represents message or description associated with the given error name.
	Message string `json:"message"`
}

// Error implements the `error` interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

// NewError returns a new Error instance.
func NewError(name, message string) *Error {
	return &Error{
		Name:    name,
		Message: message,
	}
}

var _ error = (*Error)(nil)
