# Product Requirements — HelloKit

**Status:** Draft MVP contract (not implementation)  
**Target:** Windows-first personal developer machine setup  
**Product statement:** "Choose a setup. Review what changes. Restore your tools."

## 1. Problem and users

Developer setup is fragmented across app installers, runtime versions, editor extensions and configuration files. Moving to a new computer usually requires manual recall, inconsistent versions and repeated steps.

Primary user: a developer who wants to preserve a personal setup called **Estzdev** and optionally start from an interactive **Custom** selection. Secondary user: an open-source contributor sharing a reviewable, non-secret profile.

## 2. Main user journeys

### A. New machine, personal profile

1. Download a trusted HelloKit binary on a fresh Windows machine.
2. Run `hellokit setup`, choose **Estzdev** (starter sample, editable), and inspect the profile.
3. HelloKit detects OS, architecture, package tool readiness and existing installs.
4. User sees which items are already present, missing, incompatible or **Unknown**.
5. User explicitly confirms installation of missing packages and any distinct config changes.
6. HelloKit applies and verifies each step, showing complete/partial/failed results.

### B. Custom setup

1. Run `hellokit setup`, choose **Custom**.
2. Select packages from a maintained catalog, optional runtimes and versions, and supported config restoration.
3. Review a generated local profile; optionally save it under a name.
4. Preview, confirm and apply the **same** engine used by Estzdev.

### C. Reuse a local profile

1. Run `hellokit plan --file path/to/profile.yaml`.
2. Review source and requested changes.
3. Run `hellokit setup --file path/to/profile.yaml` to apply.
4. Re-run setup; already satisfied exact requirements are skipped when known.

## 3. MVP functional requirements

| ID | Requirement | Acceptance condition |
| --- | --- | --- |
| FR-01 | Interactive setup selection | On TTY, shows Estzdev, Custom and local profile choices; cancellation has no side effects. |
| FR-02 | Local profiles | User can load and validate a local YAML profile without network/account; can list stored profiles. |
| FR-03 | Strict validation | Invalid schema version, unknown keys, duplicate IDs and unsafe paths fail before mutation. |
| FR-04 | Windows preflight | Reports OS, architecture, WinGet availability, and permission/network constraints without making changes. |
| FR-05 | Declarative plan | Stable list of intended steps with target, source, version, changes and state: satisfied / missing / unknown / blocked. |
| FR-06 | Read-only preview | `plan` and `--dry-run` make no system or config changes, including no implicit package manager bootstrap. |
| FR-07 | WinGet adapter | Installs approved exact package IDs from declared source; errors propagate and execution is logged safely. |
| FR-08 | Runtime adapter | Optional mise-based runtimes with exact target version; its prerequisite installation is included in the plan. |
| FR-09 | Config writes | Only supported, constrained config targets; diff, explicit approval, backup and post-write verification. |
| FR-10 | Safe execution | Step-level result, cancellation support and clear handling of partial success. |
| FR-11 | Re-runnable | No unnecessary repeated mutation for an already-satisfied requirement; unknown state never silently skipped. |
| FR-12 | Reports | Human-readable completion report and failure reasons with appropriate exit status. |
| FR-13 | Non-TTY path | All operations possible with explicit profile source and flags; operations requiring new consent fail cleanly. |

### Non-functional requirements

- **Offline-first:** Local profile validation and planning of purely local facts work offline. Download-based installs require connectivity and report network failures.
- **Predictability:** No silent package upgrades, forced reboots, uninstallation or config replacement.
- **Portability:** Ship an ordinary Windows executable without needing Go/Bun/Python installed on the target computer.
- **Observability:** Show exact steps, selected package source, outcome and suggested next action; redact credentials.
- **Performance:** Startup and read-only validation should not depend on fetching an online package catalog. No strict timing SLA is declared before measurement.
- **Compatibility:** Windows 10/11 x64 first; ARM64 support is a follow-on validation target, not automatically guaranteed.

## 4. In scope vs out of scope

**MVP in scope**
- Windows CLI, Estzdev sample, interactive Custom, local profile files.
- WinGet-installed tools and optional mise-managed runtime versions.
- Supported editor extension installation via explicit adapter when selected and when editor binary is available; no blanket assumption it works for all editors.
- Small, explicit allowlist of Git/editor/shell settings; adding a config target requires tests.
- Read-only preview, confirmation, verification, limited local reports.

**Not MVP**
- Web dashboard, login and cloud profile sync.
- Automatic capture of *all* installed apps and personal files.
- Full disk imaging, OS migration, Windows Update or driver installation.
- Arbitrary profile code execution / installers downloaded by URL.
- Project scaffolding, Git repo templates, cloud IDE setup.
- Linux/macOS and organization-wide policy management.
- Full automatic rollback of installations.

## 5. Definition of a successful first release

A new Windows VM can load an Estzdev **sample** profile, plan it, execute selected supported tools with explicit approval, verify output, and repeat setup without unexpected changes. Errors (missing WinGet, unavailable package, offline network, cancellation, non-admin context, failed verification) are tested and understandable. No stage sends profile data to a server.

## 6. Success metrics (measure after implementation)

- E2E first-machine setup completion rate in a controlled VM test matrix.
- Number of unexpected changes on a second run (target: zero for supported stable state).
- Percentage of failed steps with actionable messages.
- Zero secrets exposed via plain logs or exported local profile by default.

## 7. Open questions / decisions before coding each feature

- Confirm actual Estzdev package inventory and exact desired versions; **never invent them**.
- Which application config targets should Phase 1 support first? Git and VS Code are candidates.
- Choose package readiness strategy based on verified WinGet APIs/output on supported versions; do not rely on table spacing or locale-specific strings.
- Decide whether install failures halt by default or continue independent steps. Proposal: halt on prerequisite failure; independent packages can continue only with explicit plan policy.
- Decide a minimum Windows build / WinGet version based on VM compatibility tests.

See [PROFILE_SPEC.md](PROFILE_SPEC.md), [CLI_DESIGN.md](CLI_DESIGN.md) and [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md).
