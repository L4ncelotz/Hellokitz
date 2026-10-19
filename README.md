# HelloKit

> One command. Your dev environment.

**Status: M0, M1, and M2 Slice 1 implemented.** The CLI builds and runs
`version`, `profile validate/list/show`, `doctor`, and `plan --file` against
local YAML profiles, with WinGet read-only capability and installed-state
probes. Installation, `setup`, named `plan --profile`, and any cloud features
do **not** exist yet.

HelloKit is an offline-first CLI for recreating a chosen developer environment on a fresh Windows machine. Pick a named profile such as **Estzdev**, use a guided **Custom** flow, or load a local profile file. HelloKit checks the machine, presents a clear change plan, asks for approval, applies the changes, and verifies the result.

The goal is **a dependable setup workflow**, not another package manager. HelloKit delegates installations to proven platform tools and does not execute arbitrary profile-provided shell scripts.

## Product principles

1. **Start locally:** CLI and local profiles must work without an account or a website.
2. **Safe by default:** `plan` never mutates the system; `setup` previews and confirms changes before applying them.
3. **Repeatable, not magically identical:** pin versions when supported and report anything that cannot be reproduced exactly.
4. **No hidden execution:** package sources, configuration writes, administrator requirements, and failures must be visible.
5. **One core engine:** interactive CLI, non-interactive CLI, and future cloud profiles feed the same validated plan-and-apply pipeline.

## Command status

Implemented and verified on Windows (read-only; nothing is installed):

```powershell
hellokit version
hellokit profile validate --file examples/estzdev.profile.yaml
hellokit profile list
hellokit profile show estzdev
hellokit doctor
hellokit plan --file examples/estzdev.profile.yaml
```

Planned but not implemented:

```powershell
hellokit setup
# Select: Estzdev / Custom / Local profile
# Preflight -> Preview -> Confirm -> Apply -> Verify

hellokit plan --profile estzdev
```

The included [`examples/estzdev.profile.yaml`](examples/estzdev.profile.yaml) is a **sample** profile, not a capture of anyone's actual machine. Edit its selections before using it.

## Documentation map

| File | Purpose |
| --- | --- |
| [AGENTS.md](AGENTS.md) | Rules and verification steps for coding agents |
| [docs/PRODUCT.md](docs/PRODUCT.md) | Users, MVP requirements, scope, acceptance criteria |
| [docs/TECH_STACK.md](docs/TECH_STACK.md) | Technology choices and reasons |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Modules, data flow, failure behavior |
| [docs/CLI_DESIGN.md](docs/CLI_DESIGN.md) | Commands and terminal interaction design |
| [docs/DESIGN.md](docs/DESIGN.md) | Brand, terminal UX patterns and future web UI direction |
| [docs/PROFILE_SPEC.md](docs/PROFILE_SPEC.md) | Proposed v1 profile schema and validation |
| [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) | Implementation checklist, phases, definition of done |
| [docs/ROADMAP.md](docs/ROADMAP.md) | MVP vs later releases |
| [docs/SECURITY_AND_TESTING.md](docs/SECURITY_AND_TESTING.md) | Safety controls and test matrix |
| [docs/CLOUD_DESIGN.md](docs/CLOUD_DESIGN.md) | Future website / authentication / sync boundary |

## Intended first release

Windows-first native Go binary; local profile loading; Estzdev and Custom setup paths; WinGet applications; optional mise-based runtime management; plan/dry-run; confirmation; deterministic sequential application; verification and diagnostic reports.

**Out of scope for the MVP:** account registration, cloud sync, a full website, Linux/macOS adapters, arbitrary scripts, unattended OS configuration, and installing by URL.

## Development status

M0 bootstrap, M1 read-only slices, and M2 Slice 1 (WinGet read-only adapter
and typed inventory observations) are implemented and verified: canonical Go
module (`github.com/L4ncelotz/Hellokitz`), MIT license, Cobra CLI, strict
schema v1 profile loading with local/builtin resolution, `doctor` host checks,
column-anchored WinGet installed lookup with HRESULT support, and read-only
`plan --file` previews.
No installer, package installation code, `setup` flow, or remote features
exist. See [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) for
remaining milestones.

## Development & Build

Requires Go 1.24+. All commands run from the repository root.

```powershell
go test ./...
go build -o hellokit.exe ./cmd/hellokit
.\hellokit.exe --help
.\hellokit.exe version
.\hellokit.exe doctor
.\hellokit.exe profile validate --file examples/estzdev.profile.yaml
.\hellokit.exe profile list
.\hellokit.exe plan --file examples/estzdev.profile.yaml
```

## References

- [AGENTS.md conventions](https://agents.md/)
- [Cobra](https://github.com/spf13/cobra)
- [Huh](https://github.com/charmbracelet/huh)
- [WinGet documentation](https://learn.microsoft.com/en-us/windows/package-manager/winget/)
- [mise installation and Windows support](https://mise.jdx.dev/installing-mise.html)
