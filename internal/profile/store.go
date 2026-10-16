package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// SourceLocal labels a user-owned profile file.
	SourceLocal = "local"
	// SourceBuiltin labels a bundled read-only sample profile.
	SourceBuiltin = "builtin"
	// BuiltinPrefix namespaces bundled samples (e.g. "builtin:estzdev").
	BuiltinPrefix = "builtin:"
)

// Builtin is one bundled read-only sample profile document.
type Builtin struct {
	Data []byte
}

// Store resolves named profiles from a user-owned directory plus bundled
// read-only samples. It never writes to the machine.
type Store struct {
	// UserDir holds user-owned *.yaml / *.yml profiles. Empty means none.
	UserDir string
	// Builtins are bundled read-only sample documents.
	Builtins []Builtin
}

// Entry describes one discovered profile.
type Entry struct {
	// Name is the declared profile name; empty when the document is invalid.
	Name string
	// File is the local file name; empty for builtins.
	File string
	// Source is SourceLocal or SourceBuiltin.
	Source string
	// Path is the local file path; empty for builtins.
	Path string
	// Profile is the validated profile; nil when Err != nil.
	Profile *Profile
	// Err is the load/validation failure; nil for valid entries.
	Err error
	// Collision is true when another user-owned file claims the same
	// case-insensitive name.
	Collision bool
}

// Valid reports whether the entry parsed and validated successfully.
func (e Entry) Valid() bool { return e.Err == nil }

// List returns every discoverable profile, sorted by case-insensitive name
// then source. A missing user directory yields the builtins only; invalid
// documents are reported as entries (never silently dropped).
func (s Store) List() ([]Entry, error) {
	var entries []Entry

	for _, b := range s.Builtins {
		p, err := Parse(b.Data)
		e := Entry{Source: SourceBuiltin, Err: err}
		if err == nil {
			e.Name = p.Name
			e.Profile = p
		}
		entries = append(entries, e)
	}

	if s.UserDir != "" {
		files, err := os.ReadDir(s.UserDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("cannot list profiles in %q: %w", s.UserDir, err)
			}
		} else {
			for _, f := range files {
				if f.IsDir() || !isYAMLFile(f.Name()) {
					continue
				}
				path := filepath.Join(s.UserDir, f.Name())
				p, err := LoadFile(path)
				e := Entry{File: f.Name(), Source: SourceLocal, Path: path, Err: err}
				if err == nil {
					e.Name = p.Name
					e.Profile = p
				}
				entries = append(entries, e)
			}
		}
	}

	markUserCollisions(entries)
	sort.Slice(entries, func(i, j int) bool {
		ni, nj := strings.ToLower(entries[i].DisplayName()), strings.ToLower(entries[j].DisplayName())
		if ni != nj {
			return ni < nj
		}
		if entries[i].Source != entries[j].Source {
			return entries[i].Source < entries[j].Source
		}
		return entries[i].File < entries[j].File
	})
	return entries, nil
}

// DisplayName returns the declared profile name, falling back to the file
// name for invalid documents.
func (e Entry) DisplayName() string {
	if e.Name != "" {
		return e.Name
	}
	if e.File != "" {
		return e.File
	}
	return "(invalid builtin)"
}

func isYAMLFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

// markUserCollisions flags valid user-owned entries whose case-insensitive
// name is claimed by more than one user file.
func markUserCollisions(entries []Entry) {
	counts := make(map[string]int)
	for _, e := range entries {
		if e.Source == SourceLocal && e.Valid() {
			counts[strings.ToLower(e.Name)]++
		}
	}
	for i := range entries {
		if e := &entries[i]; e.Source == SourceLocal && e.Valid() && counts[strings.ToLower(e.Name)] > 1 {
			e.Collision = true
		}
	}
}

// Resolve returns the profile for name, following PROFILE_SPEC precedence:
// a "builtin:" prefixed name selects bundled samples only; otherwise a
// user-owned profile wins over a bundled sample of the same
// (case-insensitive) name, and multiple user-owned claimants are an
// ambiguity error. Invalid documents are reported, never skipped silently.
func (s Store) Resolve(name string) (*Profile, Entry, error) {
	if name == "" {
		return nil, Entry{}, fmt.Errorf("profile name is empty")
	}
	entries, err := s.List()
	if err != nil {
		return nil, Entry{}, err
	}

	if rest, ok := strings.CutPrefix(name, BuiltinPrefix); ok {
		for _, e := range entries {
			if e.Source == SourceBuiltin && strings.EqualFold(e.Name, rest) {
				if !e.Valid() {
					return nil, e, fmt.Errorf("bundled profile %q is invalid: %w", e.Name, e.Err)
				}
				return e.Profile, e, nil
			}
		}
		return nil, Entry{}, fmt.Errorf("no bundled profile named %q", rest)
	}

	var local []Entry
	for _, e := range entries {
		if e.Source == SourceLocal && e.Valid() && strings.EqualFold(e.Name, name) {
			local = append(local, e)
		}
	}
	if len(local) > 1 {
		paths := make([]string, len(local))
		for i, e := range local {
			paths[i] = e.Path
		}
		return nil, Entry{}, fmt.Errorf("profile name %q is claimed by multiple files: %s; rename one file or use --file",
			name, strings.Join(paths, ", "))
	}
	if len(local) == 1 {
		return local[0].Profile, local[0], nil
	}
	for _, e := range entries {
		if e.Source == SourceBuiltin && strings.EqualFold(e.Name, name) {
			if !e.Valid() {
				return nil, e, fmt.Errorf("bundled profile %q is invalid: %w", e.Name, e.Err)
			}
			return e.Profile, e, nil
		}
	}
	return nil, Entry{}, fmt.Errorf("no profile named %q; see `hellokit profile list`", name)
}
