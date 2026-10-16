# CLI and Terminal UX Design

**Status:** Proposed command contract / interaction design. All transcripts here are **mock examples**, not actual CLI output.

## 1. UX goals

- A first-time developer can set up a selected environment without memorizing flags.
- An experienced developer can generate a plan, inspect it and use non-interactive commands in CI-like workflows.
- Nothing is installed before users see the change plan and approve it.
- Profiles are **data**; choosing a saved profile does not grant it blanket permission to run arbitrary code.
- Screen-reader-friendly, keyboard-first interaction and readable plain-text fallback.

## 2. Commands: proposed MVP

| Command | Purpose | Mutates machine? |
| --- | --- | --- |
| `hellokit setup` | Interactive setup menu: Estzdev / Custom / file | Only after plan + approval |
| `hellokit setup --profile estzdev` | Use named profile | Only after approval |
| `hellokit setup --file profile.yaml` | Use explicit local file | Only after approval |
| `hellokit setup --file profile.yaml --dry-run` | Show planned changes (alias for read-only preview) | **No** |
| `hellokit plan --profile estzdev` | Build a read-only execution plan | **No** |
| `hellokit doctor` | Report OS / WinGet / mise / permissions status | **No** |
| `hellokit profile list` | List local and built-in available profiles | **No** |
| `hellokit profile show estzdev` | Summarize selected profile safely | **No** |
| `hellokit profile validate --file x.yaml` | Validate schema and capabilities | **No** |
| `hellokit profile create` | Guided creation; user explicitly saves a YAML profile | Writes only user-selected local file |
| `hellokit version` | Print CLI version / build metadata | **No** |

Notes:
- `--profile` and `--file` are mutually exclusive.
- `--yes` is only accepted with an explicit source and `--non-interactive` intent or a previously reviewed plan policy. It approves the **HelloKit action plan** only, not software license terms, admin elevation, or destructive config overwrites.
- Configuration write approval is separately scoped; in non-TTY mode an unsafe write must fail unless a documented explicit approval parameter is provided.
- In MVP there are **no** `login`, `sync`, `cloud` or `init-project` commands.

## 3. Interactive journey

```text
$ hellokit setup

Hello, developer!
Choose a starting point:
  > Estzdev      Starter setup profile (editable)
    Custom       Select tools and versions
    Local file   Import an existing YAML profile

Checking this Windows machine...
  [OK]    WinGet available
  [OK]    Git already installed
  [MISSING] VS Code
  [UNKNOWN] Python version cannot be verified

Planned changes (preview only)
  SKIP     Git.Git                          Already satisfied
  INSTALL  Microsoft.VisualStudioCode     Source: winget
  BLOCKED  python runtime                Verify mise prerequisite

No changes have been made.
Continue with allowed actions? [y/N]
```

Mock transcript. The **actual** tool should report current source, version, target and privilege/license warnings when applicable. Unknown/blocked items must not be automatically installed behind a simple yes prompt.

### Custom flow

1. Choose category: Applications / Runtimes / Editor extensions / Supported settings.
2. Pick items from a finite catalog. Display exact package identifier and source.
3. Select exact version or acknowledge floating version (when allowed).
4. Enter profile name (optional) and save it locally.
5. Show generated profile summary and request approval of the immutable execution plan.
6. Apply; report per-step results and instructions to restart terminal when necessary.

### Cancellation

- `Esc` or `Ctrl+C` before applying exits without side effects.
- Cancellation during execution stops launching **new** actions. An already-started third-party installer may finish; output must explain this.
- Never label cancellation as a successful setup.

## 4. Copy and tone

- Brand: **HelloKit** — friendly greeting, professional execution detail.
- Initial greeting: `Hello, developer!`
- Setup tagline: `Your environment. Your way.`
- Action words: `Will install`, `Will update`, `Will modify`, `Already satisfied`, `Cannot verify`, `Needs attention`.
- Avoid deceptively reassuring text such as `Everything is safe`, `Fully restored`, `Identical machine`.
- Warn plainly: `This action may require an administrator prompt` and `Installer changes may not be reversible`.

## 5. Visual conventions

| Category | Representation | Required accessibility behavior |
| --- | --- | --- |
| Ready | `[OK]` | Text label; color supplementary |
| Planned change | `[INSTALL]` / `[MODIFY]` | Clearly distinguish action from detection |
| Unknown | `[UNKNOWN]` | Do not treat as success |
| Blocked | `[BLOCKED]` | Show reason and suggested fix |
| Failed | `[FAILED]` | Include failed target and next step |

- Render cleanly on Windows Terminal and plain PowerShell console.
- Use screen-reader / accessible mode when requested; avoid essential information communicated solely by color.
- Limit animations / progress spinners; keep `--no-color` and non-interactive plain-text paths.
- When stdout is redirected or terminal is absent, do not call interactive prompts.

## 6. Reports and exit statuses (proposed)

- `0`: all requested, supported actions are satisfied and verified, or read-only plan completed successfully.
- `1`: generic execution failure / some actions failed.
- `2`: invalid input / unsupported schema / usage error.
- `3`: blocked prerequisite / missing capability / unknown state requiring review.
- `130`: interrupted by user (when supported by terminal / OS).

Exit code assignments can be amended before first stable release, but must be consistent, documented and tested. A partial setup must never exit `0` silently. Decided (M1): the full taxonomy lands with plan/setup; current read-only commands use exit `0` on success and `1` on failure.

## 7. Future web connection (not MVP)

```text
hellokit login
hellokit profile list --remote
hellokit setup --profile estzdev --remote
```

Potential future syntax only. Downloaded profiles must pass the exact same schema/security checks and preview gates as local profiles.
