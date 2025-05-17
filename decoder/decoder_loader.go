package decoder

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoadDecoderScript loads a decoder script from the specified path.
func LoadDecoderScript(name string) (string, error) {
	path := filepath.Join("decoders", name)
	script, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read decoder file: %w", err)
	}
	return string(script), nil
}
