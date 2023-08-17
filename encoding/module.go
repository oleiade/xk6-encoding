package encoding

import (
	"fmt"

	"github.com/dop251/goja"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

type (
	// RootModule is the global module instance that will create Client
	// instances for each VU.
	RootModule struct{}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		vu modules.VU

		*TextDecoder
		*TextEncoder
	}
)

// Ensure the interfaces are implemented correctly
var (
	_ modules.Instance = &ModuleInstance{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance implements the modules.Module interface and returns
// a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	vu.Runtime().SetFieldNameMapper(goja.TagFieldNameMapper("js", true))

	return &ModuleInstance{
		vu:          vu,
		TextDecoder: &TextDecoder{},
		TextEncoder: &TextEncoder{},
	}
}

// Exports implements the modules.Instance interface and returns
// the exports of the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{Named: map[string]interface{}{
		"TextDecoder": mi.NewTextDecoder,
		"TextEncoder": mi.NewTextEncoder,
	}}
}

// NewCmd is the JS constructor for the Cmd object.
func (mi *ModuleInstance) NewTextDecoder(call goja.ConstructorCall) *goja.Object {
	rt := mi.vu.Runtime()

	// Parse the label parameter
	var label string
	err := rt.ExportTo(call.Argument(0), &label)
	if err != nil {
		common.Throw(rt, NewError(RangeError, "unable to extract label from the first argument; reason: "+err.Error()))
	}

	// Parse the options parameter
	var options textDecoderOptions
	err = rt.ExportTo(call.Argument(1), &options)
	if err != nil {
		common.Throw(rt, err)
	}

	td, err := newTextDecoder(rt, label, options)
	if err != nil {
		common.Throw(rt, err)
	}

	return newTextDecoderObject(rt, td)
}

func (mi *ModuleInstance) NewTextEncoder(call goja.ConstructorCall) *goja.Object {
	rt := mi.vu.Runtime()

	var label EncodingName
	err := rt.ExportTo(call.Argument(0), &label)
	if err != nil {
		common.Throw(rt, NewError(RangeError, "unable to extract label from the first argument; reason: "+err.Error()))
	}

	te, err := newTextEncoder(label)
	if err != nil {
		common.Throw(rt, err)
	}

	return rt.ToValue(te).ToObject(rt)
}

// newTextDecoderObject converts the given TextDecoder instance into a JS object.
//
// It is used by the TextDecoder constructor to convert the Go instance into a JS,
// and will also set the relevant properties as read-only as per the spec.
//
// In the event setting the properties on the object where to fail, the function
// will throw a JS exception.
func newTextDecoderObject(rt *goja.Runtime, td *TextDecoder) *goja.Object {
	obj := rt.NewObject()

	// helper function to set a property on the object as read-only
	setReadOnlyProperty := func(name string, value interface{}) {
		if err := obj.DefineDataProperty(name, rt.ToValue(value), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE); err != nil {
			common.Throw(rt, fmt.Errorf("unable to define %s read-only property on TextDecoder object; reason: %w", name, err))
		}
	}

	setReadOnlyProperty("decode", td.Decode)
	setReadOnlyProperty("encoding", td.Encoding)
	setReadOnlyProperty("fatal", td.Fatal)
	setReadOnlyProperty("ignoreBOM", td.IgnoreBOM)

	return obj
}
