# Implementation Plan — HelloKit Windows MVP

**Status: M0 and M1 implemented and verified locally (see gates below); M2 not
started.** Checkboxes represent work to be performed, not completed
functionality. Prioritize working, safe vertical slices over building a broad
architecture upfront.

## Delivery policy

- One change should have a small verifiable outcome; don't implement all CLI commands at once.
- Every phase requires automated tests and clear manual verification steps.
- Local development must never attempt to install system packages as an incidental test.
- Prototype unsafe/fragile platform integration behind a fake adapter first, then validate on a disposable Windows VM.
- Before marking any task done, run the relevant checks and record actual output and limitations.

## M0 — Repository and specifications

- [ ] Agree on final GitHub module path, open-source license, owner and package release policy. (Decided: canonical module path is `github.com/L4ncelotz/Hellokitz` now that the repository exists. The `module hellokit` rename is deferred to a dedicated chore because it touches every import; no functional need yet. License/owner/release policy still open.)
- [ ] Decide initial supported Windows versions/architectures through VM inspection.
- [ ] Confirm actual Estzdev app list and versions; keep the included example illustrative until then.
- [ ] Initialize Go module with a supported toolchain and review/pin Cobra, Huh v2, YAML v4. (Done except Huh v2: Cobra v1.9.1 and YAML v4 prerelease pinned; Huh v2 deferred to M3 interactive work.)
- [x] Create `cmd/hellokit` entry point and minimal `internal` packages as needed.
- [x] Add README build instructions **after** verifying them in a clean checkout. (Verified via fresh clone: `go test -p 1 ./...`, build, and `version`/`doctor`/`profile`/`plan` smoke tests.)
- [ ] Add GitHub Actions for fmt/vet/unit tests/build on Windows; release pipeline deferred. (Workflow file added; first remote run pending on first PR.)

**Gate M0:** `hellokit version` and `hellokit --help` run on Windows without Node/Python/Bun installed; tests and build pass. No package installation code yet. (Met locally; CI run pending.)

## M1 — Local profile parser and read-only CLI

- [x] Implement bounded `schema_version: 1` YAML loader, typed structures and strict validation.
- [x] Implement local profile resolution, builtin sample support and visible name-collision behavior.
- [x] Implement `profile list`, `profile show`, `profile validate`.
- [x] Implement platform detection and `doctor` with read-only capability checks.
- [x] Implement `plan --file` with deterministic dummy/fake inventory for unit tests.
- [x] Cover wrong OS, duplicate IDs, unknown keys, malformed YAML, path issues and oversized inputs.

**Gate M1:** Invalid input cannot reach any adapter mutator. Commands produce useful errors even on non-Windows hosts (unsupported OS) instead of panicking. (Met: no mutators exist yet; non-Windows behavior covered by unit tests with injected host only — no real non-Windows run performed.)

## M2 — WinGet planning and safe execution

- [ ] Define adapter interface: capability probe, installed-state probe, install, verify.
- [ ] Implement WinGet exact ID/source invocation via `exec.CommandContext` argument arrays.
- [ ] Validate behavior of detection/version queries in a supported Windows VM, including localized consoles.
- [ ] Implement immutable plan model, missing/unknown/blocked outcomes and prerequisites.
- [ ] Implement readable plan output and `setup --dry-run` (strictly read-only).
- [ ] Implement approvals bound to plan fingerprint, with no silent license or privilege consent.
- [ ] Implement sequential apply runner with cancellation, logs, partial failure and post-install verification.
- [ ] Confirm repeat invocation skips already-satisfied packages when detection is reliable.

**Gate M2:** Fresh VM install one safe test package with manual approval; repeat setup verifies no unexpected mutation. Failure modes verified using fake adapters and VM.

## M3 — Preset and Custom UX

- [ ] Implement `setup` TTY entry path (Estzdev / Custom / Local file) with Huh.
- [ ] Add constrained local tool catalog for Custom (display package source and exact IDs).
- [ ] Implement profile save with collision warning, atomic write and no secret capture.
- [ ] Implement accessible, no-color and redirected/non-TTY paths.
- [ ] Implement `setup --profile` and `setup --file` with explicit flags and CI-safe behavior.
- [ ] Add progress and outcome reports while maintaining accurate exit codes.

**Gate M3:** All three entry paths yield an equivalent plan for equivalent profiles; cancel before apply makes zero changes.

## M4 — Runtimes / config / extensions (controlled expansion)

- [ ] Capability-test current Windows `mise` installation, version inspection and tool installation behavior.
- [ ] Make mise prerequisite an explicit plan action when runtimes are selected; no invisible manager installation.
- [ ] Add exact-version runtime checks and post-install verification. Handle shell reactivation/restart messaging.
- [ ] Add supported VS Code extensions via explicit adapter, with validation and editor prerequisite.
- [ ] Add a **small** allowlist of optional config changes; preview, explicit approval, backup and verification.
- [ ] Test config backup restoration separately from installed-package actions.

**Gate M4:** A selected runtime, extension and settings change each produce a precise plan, check capability, apply only with consent and verify independently. Unsupported config kinds fail safely.

## M5 — MVP stabilization and distribution

- [ ] Build signed or checksum-published Windows release artifact with reproducible release steps.
- [ ] Document WinGet prerequisite, installation instructions, release verification and trust warnings.
- [ ] Run automated unit/integration suite and Windows VM smoke matrix in `SECURITY_AND_TESTING.md`.
- [ ] Verify first run, offline behavior, non-admin mode, failed install, interruption, re-run, profile collisions.
- [ ] Test output usability for PowerShell and Windows Terminal with accessible mode.
- [ ] Review secrets/logs, dependency licenses and user-data storage.
- [ ] Mark supported and unsupported targets accurately in README; no misleading universal compatibility promises.

**Gate M5:** See definition of done below.

## Definition of done (MVP)

- [ ] A new eligible Windows VM can run HelloKit without a preinstalled language runtime.
- [ ] Estzdev local starter, Custom selection and explicit local file produce reviewed plans.
- [ ] Planning/dry-run is provably side-effect-free under tests.
- [ ] Supported packages, runtimes, extensions and approved config edits apply and verify as claimed.
- [ ] Re-running a satisfied profile causes no unexpected changes.
- [ ] Unsupported/unavailable/unknown items receive actionable blocked/error outcomes.
- [ ] Partial failure does not return success, invent rollback or hide skipped steps.
- [ ] No unrequested cloud calls, remote telemetry or profile-upload actions.
- [ ] Public docs correspond to actual behavior; release binary and checksums verified on fresh machine.

## Current work queue

M0 and M1 are done (pending remote CI on first PR). Start **M2** with a small,
testable slice: WinGet read-only capability and installed-state probes only —
no installation or mutation. Do not create a web app or cloud backend during
the CLI MVP. If M4 scope threatens reliability, cut optional config/editor-extension adapters from the first beta while retaining explicit unsupported status and an updated Product/README contract.

## Suggested agent prompt

> Read AGENTS.md, PRODUCT.md, ARCHITECTURE.md, PROFILE_SPEC.md and IMPLEMENTATION_PLAN.md. Implement only the first unchecked milestone that can be completed as a safe vertical slice. Do not run real installers outside a disposable Windows VM. Add tests and report the precise commands run and failures. Do not build features in the Future roadmap.
