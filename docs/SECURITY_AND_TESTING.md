# Security, Reliability and Verification

**Status:** Security acceptance rules for the proposed design; actual tests do not yet exist.

## 1. Threat model

**Assets:** Personal config files, auth material in adjacent files, machine integrity, installed executables, installer permissions, downloaded profile contents.

**Trust boundaries:** local user-provided YAML; package source metadata; external child processes; filesystem paths; downloaded cloud profiles in future.

**Attacker examples:** malicious YAML from a public repository, a profile with path traversal, compromised package manifest, unsafe subprocess arguments, a misleading config diff, a future compromised cloud profile.

## 2. Mandatory controls

| Risk | Required control | Test |
| --- | --- | --- |
| Shell injection | Fixed trusted executable + argv; no shell interpolation or YAML commands | Metacharacters in names/IDs are rejected; no external command unexpectedly launched |
| Profile parser abuse | File size cap, strict schema, duplicate/alias handling, reasonable array limits | Oversized, nested, malformed, duplicate and unknown-key inputs |
| Path traversal | Typed config targets, normalize path, reject symlinks/junction escapes | `..`, UNC, junction, symlink, sensitive dirs |
| Accidental overwrite | Scoped diff, explicit approval, verified backup, safe replacement | Existing file unchanged after reject or failure |
| Secrets leakage | No secret capture/upload, logs redact sensitive material and do not dump env | Secret sentinel not found in captured logs/exports |
| Untrusted remote content | Future cloud profiles remain declarative; same local validation and consent | Malformed/hostile remote fixture produces no mutation |
| Installation surprise | Exact package ID/source; show version/scope/license prompts | Source mismatch and wrong package ID blocked |
| Undetected failure | Verify each step's desired state; report partial failures | Fake exit 0 but no installed tool -> failure |
| Permission escalation | No automatic elevation | Non-admin denied operation is blocked with remediation |
| Drift and inconsistent state | Immutable reviewed plan + revalidate before apply; runner lock | Concurrent runs and changed inventory fail/replan appropriately |
| Network outage | Download steps fail clearly; local plan remains usable | Offline VM and fake network timeout |
| Cancellation | Context cancellation; no new steps launched after cancel | Cancel before/during install, clear partial result |

## 3. Default policy

- Never auto-accept third-party EULAs/license prompts merely due to HelloKit confirmation. Model separate package-manager acceptance explicitly where supported.
- No private-key, SSH config secret, credentials, corporate configuration, full environment-variable export or browser data capture by default.
- No background daemon, update service, remote telemetry or listening TCP port in MVP.
- No silent upgrades, uninstalls, reboots or changes to Windows Security/UAC policies.
- Profiles from web, GitHub or local disk are never considered trusted merely because a user selected them.
- Include clear provenance and checksum validation for downloadable HelloKit releases; signing is preferred when a release process supports it.

## 4. Required automated testing pyramid

### Unit (cross-platform with fakes)

- Parser: schema versions, empty file, malformed/oversized YAML, duplicates, unknown keys, wrong platform, invalid values.
- Planning: supported/installed/missing/unknown/blocked; dependency ordering; deterministic output; version mismatch; floating version warning.
- Runner: fake subprocess success/failure, context cancellation, timeout, partial results, status propagation and post-verification failure.
- Filesystem: safe target resolution, diffs, backups, failed atomic writes, symlink and traversal escape, lock behavior.
- CLI: wrong flags, mutually exclusive sources, prompts gated by TTY, no-color mode, accurate exit codes.
- **Mutation sentinel:** instrument mutating adapter; assert zero calls from `plan`, `doctor` or `--dry-run`.

### Integration (process-level fakes)

- Inject a fake WinGet executable to exercise exact argv construction, quoting and error handling.
- Inject a fake mise executable to verify prerequisite and exact version handling.
- Test compatibility with programmatic cancellation and simulated partial install.
- Exercise equivalent plans across interactive, file and named-profile paths.

### Real Windows VM E2E (manual or isolated CI runner)

| Scenario | Expected |
| --- | --- |
| Fresh eligible Windows + WinGet | Successful preflight / supported package plan |
| Windows without ready WinGet | Actionable blocked status; no implicit installer bootstrap |
| Same profile, second run | Satisfied items skipped; no unexpected mutation |
| Limited permissions | Clear blocked or expected elevation warning, no hidden UAC |
| Offline network | Local read-only commands work; download steps fail gracefully |
| Bad package ID / source | Never install another package by fuzzy name |
| Locale changes | Inventory does not guess based on translated column spacing |
| Interrupted installer | Partial status and no new actions after cancel |
| Config diff rejected | Original file remains byte-identical |
| Unsupported architecture | Refuse apply; read-only diagnostics remain available |

## 5. Verification commands (once code exists)

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/hellokit
```

The final executable and integration commands must be run on a Windows host/VM. Never claim that running Go unit tests on Linux proves WinGet integration.

## 6. Release gate

No public executable release without: verified build hashes, documented supported WinGet/mise behavior, tested cancellation, first-run and second-run VM success, profile input fuzz/regression cases, and a clear statement that third-party package installation is not generally rollback-safe.
