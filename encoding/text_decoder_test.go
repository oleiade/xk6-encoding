package encoding

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
)

//
// [WPT test]: https://github.com/web-platform-tests/wpt/blob/b5e12f331494f9533ef6211367dace2c88131fd7/encoding/textdecoder-labels.any.js
func TestTextDecoder(t *testing.T) {
	t.Parallel()

	ts := newTestSetup(t)
	digestTestScript, err := compileFile("./tests", "textdecoder-labels.js")
	assert.NoError(t, err)

	gotScriptErr := ts.ev.Start(func() error {
		_, err := ts.rt.RunProgram(digestTestScript)
		return err
	})

	assert.NoError(t, gotScriptErr)
}

func executeTestScripts(rt *goja.Runtime, base string, scripts ...string) error {
	for _, script := range scripts {
		program, err := compileFile(base, script)
		if err != nil {
			return err
		}

		if _, err = rt.RunProgram(program); err != nil {
			return err
		}
	}

	return nil
}
