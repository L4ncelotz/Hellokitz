# AGENTS.md — HelloKit

These are repository-wide instructions for any coding agent contributing to HelloKit.

## Read first, in order

1. `README.md` — project state and documentation index.
2. `docs/PRODUCT.md` — confirmed user goals, requirements and exclusions.
3. `docs/ARCHITECTURE.md` — required boundaries and runtime contract.
4. `docs/PROFILE_SPEC.md` and `docs/CLI_DESIGN.md` — user-visible interfaces.
5. `docs/IMPLEMENTATION_PLAN.md` — current implementation tasks.
6. `docs/SECURITY_AND_TESTING.md` — risks and mandatory verification.

**Do not treat proposed CLI output in these documents as implemented behavior.** Inspect the actual repository before making claims about features or passing tests.

## Mission and scope

Build a dependable, Windows-first, offline-first Go CLI that recreates a declared development setup from a reviewed local profile. It must never require an account, backend or browser for local profiles. Estzdev is a user-editable example preset, **not a source of unconfirmed personal settings**.

## Engineering rules

- Work incrementally according to `docs/IMPLEMENTATION_PLAN.md`; finish a vertical slice with tests before starting the next one.
- Prefer standard-library Go and small interfaces; do not introduce dependency injection frameworks, service locators, databases or speculative abstractions.
- Keep command handlers thin. Profile parsing, planning, execution and verification belong in separately testable packages.
- Separate **read-only detection** from **mutating execution**. Running `plan`, `doctor` or `--dry-run` must not install packages or write user configuration.
- Treat YAML / imported profiles / future remote profiles as untrusted input. Strictly validate schema, package IDs, paths and bounded input sizes.
- Never invoke PowerShell, `cmd /c`, `sh -c`, or other shells by interpolating profile values into command strings. Use parameterized argument arrays for known binaries.
- In MVP, reject arbitrary commands, lifecycle hooks, remote scripts, custom installer URLs, encoded payloads, or privilege escalation requests in profiles.
- Do not silently install/upgrade applications, alter PATH or shell startup files, accept third-party licenses, or require administrator elevation.
- Preserve user settings: show a diff and explicit write approval; make a backup before modifying supported configuration files.
- Do not claim full rollback of package manager operations. Config-file restore and best-effort package retry are separate capabilities.
- If installation state cannot be determined reliably, display **Unknown**, require manual resolution, and never assume **Installed**.
- Track actual process exit status and post-install verification. A zero exit code alone is not proof that the requested state was reached.
- Do not log secrets, tokens, environment-variable contents, personal paths unnecessarily, or full configuration file contents.
- Never commit real Estzdev personal credentials or corporate settings to the public repository.
- Dependencies and current external CLI behavior must be verified against official documentation before new integration is merged.
- When requirements conflict or remain ambiguous, record the decision in the relevant spec or open question rather than guessing a feature into existence.

## Suggested code layout (create only as needed)

```text
cmd/hellokit/           executable entry point
internal/cli/           Cobra commands and UI adapter
internal/profile/       read, validate, resolve local profiles
internal/plan/          inventory + immutable execution plan
internal/runner/        apply plan, errors, retries, verification
internal/adapters/      winget, mise, config-file adapters
internal/platform/      Windows host checks and filesystem paths
internal/report/        user-readable and structured outcome reports
```

Use module path chosen when the GitHub repository is created; **do not invent a final repository URL**.

## Development commands (when Go module exists)

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/hellokit
```

The race test may require compatible toolchain / CGO. On a non-Windows CI runner, isolate Windows-dependent adapters and run unit tests with fakes. Real install smoke tests must run in disposable Windows VMs, never automatically on a contributor's workstation.

## Before completing any task

1. State changed files and the root cause / intent.
2. Add focused tests for success, failure, cancellation and repeat invocation.
3. Run relevant commands and report the **actual** results; do not report unrun checks as passed.
4. Check docs, public flags, schema compatibility and safety requirements.
5. Update `docs/IMPLEMENTATION_PLAN.md` checkbox status **only after verification**.

## Git Workflow (Mandatory)

All completed changes MUST be committed and pushed to the remote Git repository automatically.

### Rules

1. **Branch Strategy**
   - `feat/<description>` — New features
   - `fix/<description>` — Bug fixes
   - `hotfix/<description>` — Critical fixes
   - `refactor/<description>` — Code improvements
   - `docs/<description>` — Documentation
   - `chore/<description>` — Maintenance

2. **Before Implementation**
   - Check `git status` and the current branch.
   - Create a task-specific branch from the latest main branch.
   - Preserve existing uncommitted changes.
   - Never overwrite or discard unrelated work.

3. **After Implementation**
   - Run relevant tests, linting, and build verification.
   - Review the diff for correctness and unintended changes.
   - Stage only files related to the task.
   - Use Conventional Commits (`feat:`, `fix:`, `refactor:`, `docs:`, etc.).
   - Commit and push the task branch to `origin`.
   - Create a Pull Request targeting `main` when supported.

4. **Safety**
   - Never commit secrets, credentials, `.env` files, or private keys.
   - Never force-push or push directly to `main`.
   - Never automatically merge Pull Requests.
   - If verification fails, fix the issue before pushing.
   - If Git authentication, permissions, or PR creation fails, report the blocker instead of claiming success.

5. **Completion Report**

   Every completed task must report:
   - Changes made
   - Tests and verification results
   - Branch name
   - Commit hash
   - Push status
   - Pull Request URL (if created)

**Default behavior:** Do not stop after making local changes. Complete the Git workflow automatically without requiring an additional instruction, provided verification passes and repository access is available.

## MVP boundaries

Do not add cloud login, website, telemetry, background agents, automatic self-update, cross-platform installers, arbitrary shell plugins, project generators or container orchestration unless expressly requested as the next phase.
