# Tech Stack and Selection Rationale

**Status:** Proposed selections for the first Windows MVP. Selection is based on task fit, reliability, distribution and lifetime maintenance—not personal familiarity with a language.

## 1. Chosen MVP technologies

| Area | Choice | Why | Notes |
| --- | --- | --- | --- |
| CLI executable | **Go** | Single native binary, mature Windows process/file APIs, straightforward testing and cross-compilation | Exact Go toolchain version pinned when implementation begins |
| Subcommands / flags | **Cobra** (`github.com/spf13/cobra`) | Proven CLI command hierarchy, help/completions | Keep handlers thin; no need for Viper by default |
| Interactive forms | **Huh** (`charm.land/huh/v2`) | Select/multiselect/confirm, screen-reader accessible mode | Interactive layer optional; deterministic non-TTY execution must work |
| YAML codec | **go.yaml.in/yaml/v4** (pinned `v4.0.0-rc.6` prerelease) | Maintained upstream under YAML organization | Strict schema decoding plus separate semantic checks. **Known risk:** v4 has no stable release yet; the prerelease is pinned and must not be changed without justification. Revisit when a stable v4 ships. |
| Core platform integration | **os/exec + Go standard library** | Explicit child processes and filesystem safety; no heavyweight framework | Pass argument arrays, never concatenate shell commands |
| Windows packages | **WinGet** | OS-native ecosystem and exact package IDs | Verify behavior across supported versions; no own package manager |
| Toolchains / runtimes | **mise** | One version manager for multiple languages with current Windows support | Adapter only; runtime support must be capability-tested |
| Testing | Go `testing`, fakes + disposable Windows VMs | Fast deterministic unit tests plus real platform validation | Never run real install tests on a contributor machine |
| Local state | User configuration/cache directories + YAML/JSON reports | No DB or daemon required | Atomic file replacement where practical |

## 2. Tradeoffs: Go vs Rust vs Bun vs C#

| Candidate | Advantages | Costs | Verdict |
| --- | --- | --- | --- |
| Go | Easy distribution, small operational surface, readable CLI orchestration | Less compile-time control of memory than Rust | **Best practical fit for MVP** |
| Rust | Strong memory safety and fine-grained system integration | Higher implementation/review complexity for orchestration-first code | Revisit only if required by real performance/security constraints |
| TypeScript + Bun | Fast iteration for JS ecosystem developers | Adds runtime/packaging choices and extra Windows distribution checks | Not selected for initial self-bootstrapping CLI |
| C#/.NET | Excellent Windows APIs | Self-contained distribution and targeting details; increases Windows affinity | Valid Windows-only alternative, less aligned with possible later platforms |
| Python | Automation libraries and quick prototyping | Packaging and runtime bootstrap complexity | Not selected |

**Constraint:** Future changes must be justified with benchmarks, implementation evidence or new product requirements—not personal preference.

## 3. Architectural choices

- No application backend, database, auth, browser, always-running service or telemetry in the MVP.
- Do not automatically install runtime manager to run HelloKit itself; Go binary works standalone.
- WinGet is used as an **adapter**, not as an internal dependency of plan parsing.
- mise may be installed as an **explicitly visible prerequisite** if a chosen profile requires runtimes.
- Prefer Go stdlib `log/slog` for local diagnostic events; default console output stays readable and secrets must be redacted.
- Avoid OS-specific string parsing where machine-readable APIs are available. Where WinGet output is insufficiently stable, show `unknown`, test the adapter against supported versions, and never infer success from fragile formatting.

## 4. Future website (NOT an MVP dependency)

Initial architectural candidate: **SvelteKit + TypeScript** for web views and server endpoints, **PostgreSQL** for user metadata and revisioned profiles, standards-based **OIDC/OAuth** for login, and HTTPS API for the CLI. The need for a separate Go service, queue, Redis or object storage is **not established** and should not be presumed. Re-evaluate at Phase 3.

See [CLOUD_DESIGN.md](CLOUD_DESIGN.md) for remote trust, revisions and the separation between downloaded data and code execution.

## 5. Version policy

- Pin dependencies and record checksums in released builds; upgrade intentionally via PR.
- Record supported Go release and dependency versions in `go.mod` / release notes once code exists; no unverified version numbers in the design contract.
- Treat `latest` as **non-reproducible**. For an exact environment, users must specify exact tool versions supported by their underlying managers; otherwise clearly label the result as best effort.
- CI should test every supported OS/architecture combination and package manager version range claimed in documentation.

## 6. Verified reference documentation (checked October 2026)

- Cobra: https://github.com/spf13/cobra
- Huh v2: https://github.com/charmbracelet/huh
- YAML v4 upstream: https://github.com/yaml/go-yaml
- WinGet `install`: https://learn.microsoft.com/en-us/windows/package-manager/winget/install
- WinGet `list`: https://learn.microsoft.com/en-us/windows/package-manager/winget/list
- mise Windows installation: https://mise.jdx.dev/installing-mise.html
- mise getting started: https://mise.jdx.dev/getting-started.html
