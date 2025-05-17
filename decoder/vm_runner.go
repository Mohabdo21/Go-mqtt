package decoder

import (
	"fmt"

	"github.com/dop251/goja"
)

type DecodedData struct {
	Temperature float64
	Humidity    float64
	CO2         float64
}

// RunDecoder executes the decoder script with the provided hex payload.
func RunDecoder(script string, hexPayload string) (*DecodedData, error) {
	vm := goja.New()

	if err := vm.Set("hexPayload", hexPayload); err != nil {
		return nil, fmt.Errorf("vm variable set error: %w", err)
	}

	_, err := vm.RunString(script + "\nconst output = decode(hexPayload); output;")
	if err != nil {
		return nil, fmt.Errorf("VM execution error: %w", err)
	}

	res := vm.Get("output")

	if res == nil {
		return nil, fmt.Errorf("no output returned from decoder")
	}

	var result DecodedData
	if err := vm.ExportTo(res, &result); err != nil {
		return nil, fmt.Errorf("result export error: %w", err)
	}

	return &result, nil
}
