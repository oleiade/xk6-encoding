// Package encoding is a k6 module that provides implementations of encoding.TextDecoder and encoding.TextEncoder.
package encoding

import (
	"github.com/oleiade/xk6-encoding/encoding"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/encoding", new(encoding.RootModule))
}
