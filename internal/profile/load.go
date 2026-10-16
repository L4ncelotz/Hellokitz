package profile

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoadFile reads, size-bounds, parses, and validates the profile at path.
// It performs canonical path checks (clean, existing regular file) and never
// writes to the machine.
func LoadFile(path string) (*Profile, error) {
	if path == "" {
		return nil, fmt.Errorf("profile file path is empty")
	}
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		return nil, fmt.Errorf("cannot stat profile file %q: %w", clean, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("profile file %q is not a regular file", clean)
	}
	if info.Size() > MaxProfileBytes {
		return nil, fmt.Errorf("profile file %q exceeds %d bytes", clean, MaxProfileBytes)
	}
	data, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("cannot read profile file %q: %w", clean, err)
	}
	p, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("invalid profile file %q: %w", clean, err)
	}
	return p, nil
}
