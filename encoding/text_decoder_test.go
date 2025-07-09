package encoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//
// [WPT test]: https://github.com/web-platform-tests/wpt/blob/b5e12f331494f9533ef6211367dace2c88131fd7/encoding/textdecoder-labels.any.js
func TestTextDecoder(t *testing.T) {
	t.Parallel()
	scripts := []testScript{
		{base: "./tests", path: "textdecoder-arguments.js"},
		{base: "./tests", path: "textdecoder-byte-order-marks.js"},
		{base: "./tests", path: "textdecoder-copy.js"},
		{base: "./tests", path: "textdecoder-eof.js"},
		{base: "./tests", path: "textdecoder-fatal.js"},
		{base: "./tests", path: "textdecoder-ignorebom.js"},
		{base: "./tests", path: "textdecoder-labels.js"},
	}

	ts := newTestSetup(t)
	err := executeTestScripts(ts, scripts)
	require.NoError(t, err)
}
