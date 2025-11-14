package tim

import "github.com/Snider/Enchantrix/pkg/trix"

// DefaultSpec returns a default runc spec.
func defaultConfig() (*trix.Trix, error) {
	return &trix.Trix{
		Header: make(map[string]interface{}),
	}, nil
}
