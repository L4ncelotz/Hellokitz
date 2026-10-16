# Profile Format — Proposal for Schema v1

**Status:** Draft user-visible data contract. Needs parser/validator implementation and compatibility tests before release.

## 1. Goals

- Human-reviewable, portable and versioned local YAML.
- Declarative desired state; no arbitrary shell commands or URL-based installers.
- Strict parsing of unknown fields and types.
- Compatible with future cloud delivery **without executing untrusted code**.

## 2. Example

See [`../examples/estzdev.profile.yaml`](../examples/estzdev.profile.yaml). Example below illustrates supported conceptual fields, **not necessarily the author's real setup**.

```yaml
schema_version: 1
name: estzdev
description: Starter Windows developer setup
platform:
  os: windows
  architectures: [amd64]

applications:
  - id: Git.Git
    source: winget
  - id: Microsoft.VisualStudioCode
    source: winget

runtimes: []
editor_extensions: []
settings: []
```

A later explicitly chosen runtime example:

```yaml
runtimes:
  - name: node
    manager: mise
    version: '24.11.1' # Illustrative version: confirm actual availability before use
```

Do not silently infer runtime versions or install Node, Python or Bun based on a profile name. If exact requested version is unavailable, report an actionable failure.

## 3. Schema fields

| Field | Type | Required | Validation |
| --- | --- | --- | --- |
| `schema_version` | integer | Yes | Must be exactly `1` for parser v1 |
| `name` | string | Yes | `[A-Za-z0-9][A-Za-z0-9_-]{1,39}` (2–40 chars; uppercase accepted, compared case-insensitively for portable, Windows-safe collision handling); reject conflicting names |
| `description` | string | No | Plain text, <= 240 UTF-8 bytes |
| `platform.os` | string | Yes | `windows` in v1 |
| `platform.architectures` | string array | Yes | Nonempty subset of supported architectures, initially `amd64` |
| `applications` | array | No | Each source + exact ID unique; max 100 entries |
| `applications[].id` | string | Yes | Valid WinGet identifier (limited alphabet, reasonable length); no parameters / shell punctuation |
| `applications[].source` | string | Yes | `winget` for MVP; source means WinGet community repository name, not arbitrary URL |
| `applications[].version` | string | No | Exact version when manager supports it; omitted means floating and must be disclosed |
| `runtimes` | array | No | Each `(manager, name)` unique; max 30 entries |
| `runtimes[].manager` | string | Yes | `mise` only in MVP |
| `runtimes[].name` | string | Yes | Allowlisted catalog entry (v1: `node`, `python`, `bun`), not arbitrary plugin URL. Catalog additions are backward-compatible; removals require a schema version bump |
| `runtimes[].version` | string | Yes | Exact, manager-supported version, required; reject `latest` and `*` (floating versions are not reproducible) |
| `editor_extensions` | array | No | Supported editor (v1: `vscode`) and extension identifier only; phase-gated implementation. Allowlist additions are backward-compatible |
| `settings` | array | No | Known setting kind (v1: `git.config`) + typed values; **no raw file path** or unreviewed script. Allowlist additions are backward-compatible |

**Schema discipline:** reject unknown root/nested keys; reject duplicate YAML mapping keys when codec supports detection; reject aliases or recursively constructed documents exceeding parser limits; avoid polymorphic `any`/map input in execution layer. Limit total file size (proposed 1 MiB) before parsing.

## 4. Editor extension field design (phase-gated)

```yaml
editor_extensions:
  - editor: vscode
    id: esbenp.prettier-vscode
```

Validate editor and extension identifier; ensure selected editor exists or is included in the plan; use supported CLI invocation with explicit arguments. Installing an extension is a separate visible action.

## 5. Config setting field design (phase-gated)

```yaml
settings:
  - kind: git.config
    key: init.defaultBranch
    value: main
```

**Never** represent settings as a free-form shell command. The `git.config` handler must restrict known keys, consider scope and consent, show diff, back up existing config, and verify after write. Sensitive values such as tokens, credentials, proxy passwords, private keys and company data are forbidden.

## 6. Loading and collision precedence

- `--file` loads exactly the explicit local file after canonical path checks; do not search remote sources.
- `--profile <name>` resolves user-owned saved profile first; bundled samples are separately namespaced/resolved (e.g., `builtin:estzdev`) if a name collision exists, and collisions must be displayed rather than silent. Name matching is case-insensitive so collisions behave identically on case-sensitive and case-insensitive filesystems (Windows-safe). A user-owned profile wins over a bundled sample of the same name; two user-owned files claiming one name is an ambiguity error, never a silent pick.
- `hellokit profile list` labels profiles as `local` vs `builtin`.
- Schema `1` cannot be extended with incompatible semantics without a new schema version / explicit migration.
- Profiles from future remote API follow identical validations. Cloud cannot inject commands or silently override local files.

## 7. Trust and version guarantees

The profile states **desired tools**, not a byte-for-byte machine image. Software distribution, architecture, source availability, licensing, install scope and transitive dependencies may differ. Pinning a version improves predictability but does not guarantee exact installed bits without artifact verification from upstream tools.

## 8. Implementation checklist

- [ ] Define Go typed structs with validation.
- [ ] Enforce schema version, unknown fields, uniqueness, input size and bound lengths.
- [ ] Decide supported v1 semantic defaults in tests, not implicit runtime behavior.
- [ ] Test traversal, malformed/duplicate mapping keys, oversized input, unsupported platform.
- [ ] Test cloud-compatible trust boundary with fake untrusted profile (without actual cloud).
