// Package profile reads, validates, and resolves HelloKit local profiles.
//
// Profiles are untrusted input: every document is size-bounded before parsing,
// decoded strictly (unknown fields, duplicate keys, and aliases rejected),
// and checked against the schema v1 semantic rules from docs/PROFILE_SPEC.md.
// Nothing in this package mutates the machine.
package profile

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v4"
)

const (
	// SchemaVersion is the only schema_version accepted by this parser.
	SchemaVersion = 1
	// MaxProfileBytes bounds the raw document size before parsing (1 MiB).
	MaxProfileBytes = 1 << 20
	// MaxApplications bounds the applications list length.
	MaxApplications = 100
	// MaxRuntimes bounds the runtimes list length.
	MaxRuntimes = 30
	// MaxDescriptionBytes bounds the description field (UTF-8 bytes).
	MaxDescriptionBytes = 240
	// MaxIDLength bounds package/extension identifier length.
	MaxIDLength = 128
	// MaxVersionLength bounds version string length.
	MaxVersionLength = 64
)

var (
	namePattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,39}$`)
	idPattern         = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
	settingKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

	// exactVersionPattern requires versions to start with digits (or optional 'v'/'V' prefix)
	// followed by dot-separated numeric segments and optional prerelease/build metadata.
	exactVersionPattern = regexp.MustCompile(`^[vV]?[0-9]+(\.[0-9]+)*([.-][A-Za-z0-9_.-]+)?$`)
	// floatingVersionPattern flags aliases, wildcards, ranges, and floating tags like latest, lts, stable.
	floatingVersionPattern = regexp.MustCompile(`(?i)^(latest|lts.*|stable|current|nightly|system|default|node)$|[\*xX\^~><]`)
)

func isExactVersion(v string) bool {
	if floatingVersionPattern.MatchString(v) {
		return false
	}
	return exactVersionPattern.MatchString(v)
}

// supportedOS / architectures / managers / catalogs for schema v1.
var (
	supportedArchitectures = map[string]bool{"amd64": true}
	supportedSources       = map[string]bool{"winget": true}
	supportedManagers      = map[string]bool{"mise": true}
	// supportedRuntimes is the v1 allowlist of mise catalog entries.
	supportedRuntimes = map[string]bool{"node": true, "python": true, "bun": true}
	// supportedEditors is the v1 allowlist of editor identifiers.
	supportedEditors = map[string]bool{"vscode": true}
	// supportedSettingKinds is the v1 allowlist of setting kinds.
	supportedSettingKinds = map[string]bool{"git.config": true}
)

// Profile is the typed schema v1 document.
type Profile struct {
	SchemaVersion    int               `yaml:"schema_version"`
	Name             string            `yaml:"name"`
	Description      string            `yaml:"description,omitempty"`
	Platform         Platform          `yaml:"platform"`
	Applications     []Application     `yaml:"applications,omitempty"`
	Runtimes         []Runtime         `yaml:"runtimes,omitempty"`
	EditorExtensions []EditorExtension `yaml:"editor_extensions,omitempty"`
	Settings         []Setting         `yaml:"settings,omitempty"`
}

// Platform constrains the target machine.
type Platform struct {
	OS            string   `yaml:"os"`
	Architectures []string `yaml:"architectures"`
}

// Application is one desired package from a platform source.
type Application struct {
	ID      string `yaml:"id"`
	Source  string `yaml:"source"`
	Version string `yaml:"version,omitempty"`
}

// Runtime is one desired language runtime via a version manager.
type Runtime struct {
	Name    string `yaml:"name"`
	Manager string `yaml:"manager"`
	Version string `yaml:"version"`
}

// EditorExtension is one desired editor extension.
type EditorExtension struct {
	Editor string `yaml:"editor"`
	ID     string `yaml:"id"`
}

// Setting is one declarative configuration change of a known kind.
type Setting struct {
	Kind  string `yaml:"kind"`
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// Summary counts profile contents for display.
type Summary struct {
	Applications int
	Runtimes     int
	Extensions   int
	Settings     int
}

// Summarize counts the actionable entries in a validated profile.
func (p *Profile) Summarize() Summary {
	return Summary{
		Applications: len(p.Applications),
		Runtimes:     len(p.Runtimes),
		Extensions:   len(p.EditorExtensions),
		Settings:     len(p.Settings),
	}
}

// Parse decodes and validates one YAML document. It rejects empty input,
// multi-document streams, aliases, duplicate mapping keys, and unknown
// fields before applying the semantic rules below.
func Parse(data []byte) (*Profile, error) {
	if len(data) > MaxProfileBytes {
		return nil, fmt.Errorf("profile exceeds %d bytes (%d bytes)", MaxProfileBytes, len(data))
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("profile is empty")
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("profile is not valid YAML: %w", err)
	}
	if err := rejectAliases(&node); err != nil {
		return nil, err
	}
	if err := rejectDuplicateKeys(&node); err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var p Profile
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("profile does not match schema v1: %w", err)
	}
	// A second document (trailing content after ---) is not a v1 profile.
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("profile must contain exactly one YAML document")
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

func rejectAliases(n *yaml.Node) error {
	if n.Kind == yaml.AliasNode {
		return fmt.Errorf("profile must not use YAML aliases or anchors")
	}
	for _, c := range n.Content {
		if err := rejectAliases(c); err != nil {
			return err
		}
	}
	return nil
}

func rejectDuplicateKeys(n *yaml.Node) error {
	if n.Kind == yaml.MappingNode {
		seen := make(map[string]bool, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind == yaml.ScalarNode {
				if seen[k.Value] {
					return fmt.Errorf("profile contains duplicate key %q", k.Value)
				}
				seen[k.Value] = true
			}
		}
	}
	for _, c := range n.Content {
		if err := rejectDuplicateKeys(c); err != nil {
			return err
		}
	}
	return nil
}

// Validate applies the schema v1 semantic rules to an already-decoded profile.
func (p *Profile) Validate() error {
	if p.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %d, got %d", SchemaVersion, p.SchemaVersion)
	}
	if !namePattern.MatchString(p.Name) {
		return fmt.Errorf("name %q must match [A-Za-z0-9][A-Za-z0-9_-]{1,39}", p.Name)
	}
	if len(p.Description) > MaxDescriptionBytes {
		return fmt.Errorf("description exceeds %d bytes", MaxDescriptionBytes)
	}
	if err := p.Platform.validate(); err != nil {
		return err
	}
	if len(p.Applications) > MaxApplications {
		return fmt.Errorf("applications exceeds %d entries", MaxApplications)
	}
	seenApps := make(map[string]bool, len(p.Applications))
	for i := range p.Applications {
		a := &p.Applications[i]
		if !supportedSources[a.Source] {
			return fmt.Errorf("applications[%d]: source %q is not supported in v1", i, a.Source)
		}
		if !idPattern.MatchString(a.ID) {
			return fmt.Errorf("applications[%d]: id %q is not a valid package identifier", i, a.ID)
		}
		key := strings.ToLower(a.Source + "\x00" + a.ID)
		if seenApps[key] {
			return fmt.Errorf("applications[%d]: duplicate entry for id %q from %q", i, a.ID, a.Source)
		}
		seenApps[key] = true
		if len(a.Version) > MaxVersionLength {
			return fmt.Errorf("applications[%d]: version exceeds %d characters", i, MaxVersionLength)
		}
	}
	if len(p.Runtimes) > MaxRuntimes {
		return fmt.Errorf("runtimes exceeds %d entries", MaxRuntimes)
	}
	seenRuntimes := make(map[string]bool, len(p.Runtimes))
	for i := range p.Runtimes {
		r := &p.Runtimes[i]
		if !supportedManagers[r.Manager] {
			return fmt.Errorf("runtimes[%d]: manager %q is not supported in v1", i, r.Manager)
		}
		if !supportedRuntimes[strings.ToLower(r.Name)] {
			return fmt.Errorf("runtimes[%d]: runtime %q is not in the v1 catalog", i, r.Name)
		}
		if r.Version == "" {
			return fmt.Errorf("runtimes[%d]: version is required; floating versions are not reproducible", i)
		}
		if len(r.Version) > MaxVersionLength {
			return fmt.Errorf("runtimes[%d]: version exceeds %d characters", i, MaxVersionLength)
		}
		if !isExactVersion(r.Version) {
			return fmt.Errorf("runtimes[%d]: version %q is not an exact version", i, r.Version)
		}
		key := strings.ToLower(r.Manager + "\x00" + r.Name)
		if seenRuntimes[key] {
			return fmt.Errorf("runtimes[%d]: duplicate entry for runtime %q", i, r.Name)
		}
		seenRuntimes[key] = true
	}
	for i := range p.EditorExtensions {
		e := &p.EditorExtensions[i]
		if !supportedEditors[strings.ToLower(e.Editor)] {
			return fmt.Errorf("editor_extensions[%d]: editor %q is not supported in v1", i, e.Editor)
		}
		if !idPattern.MatchString(e.ID) {
			return fmt.Errorf("editor_extensions[%d]: id %q is not a valid extension identifier", i, e.ID)
		}
		if len(e.ID) > MaxIDLength {
			return fmt.Errorf("editor_extensions[%d]: id exceeds %d characters", i, MaxIDLength)
		}
	}
	for i := range p.Settings {
		s := &p.Settings[i]
		if !supportedSettingKinds[s.Kind] {
			return fmt.Errorf("settings[%d]: kind %q is not supported in v1", i, s.Kind)
		}
		if !settingKeyPattern.MatchString(s.Key) {
			return fmt.Errorf("settings[%d]: key %q is not valid", i, s.Key)
		}
		if len(s.Value) > 4096 {
			return fmt.Errorf("settings[%d]: value exceeds 4096 characters", i)
		}
	}
	return nil
}

func (pl *Platform) validate() error {
	if pl.OS != "windows" {
		return fmt.Errorf("platform.os must be %q in v1, got %q", "windows", pl.OS)
	}
	if len(pl.Architectures) == 0 {
		return fmt.Errorf("platform.architectures must list at least one architecture")
	}
	seen := make(map[string]bool, len(pl.Architectures))
	for _, arch := range pl.Architectures {
		if !supportedArchitectures[arch] {
			return fmt.Errorf("platform.architectures entry %q is not supported in v1", arch)
		}
		if seen[arch] {
			return fmt.Errorf("platform.architectures contains duplicate entry %q", arch)
		}
		seen[arch] = true
	}
	return nil
}
