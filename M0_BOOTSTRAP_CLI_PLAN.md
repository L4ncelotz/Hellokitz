# Implementation Plan: M0 — Repository and CLI Bootstrap

## Context
HelloKit currently exists solely as a specification and documentation suite with no Go module, code, or executable. Milestone 0 establishes the Go toolchain foundation, the `cmd/hellokit` executable entry point, the Cobra-based root command and `version` subcommand, unit tests for command routing, and CI automation for Windows. Achieving Gate M0 proves that `hellokit --help` and `hellokit version` execute and pass tests on Windows with zero mutation or external runtime dependencies.

## Approach

### 1. Initialize Go Module and Toolchain
- **Target**: `go.mod`
- **Action**: Initialize Go module with `module hellokit` and Go language version `1.24` (compatible with the host's `go1.27.1 windows/amd64`).
- **Dependencies**:
  - Add Cobra: `github.com/spf13/cobra` (v1.9.1)
  - Add Pflag: `github.com/spf13/pflag` (transitive via Cobra)
- **Command**:
  ```powershell
  go mod init hellokit
  go get github.com/spf13/cobra@v1.9.1
  go mod tidy
  ```

### 2. Version Information Package
- **Target**: `internal/version/version.go`
- **Action**: Create immutable compile-time metadata package with defaults and ldflag override hooks.
- **Symbols**:
  ```go
  package version

  var (
      Version = "0.1.0-dev"
      Commit  = "none"
      Date    = "unknown"
  )

  type Info struct {
      Version string `json:"version"`
      Commit  string `json:"commit"`
      Date    string `json:"date"`
      OS      string `json:"os"`
      Arch    string `json:"arch"`
  }

  func Get() Info
  func (i Info) String() string
  ```
- **Behavior**: `Get()` resolves `runtime.GOOS` and `runtime.GOARCH` alongside the variables. `String()` formats output as `hellokit <version> (<os>/<arch>) commit: <commit> date: <date>`.

### 3. Root Command and CLI Hierarchy
- **Target**: `internal/cli/root.go`
- **Action**: Define Cobra root command and `Execute()` function.
- **Specifications**:
  - Command: `hellokit`
  - Short: `One command. Your dev environment.`
  - Long: `HelloKit is an offline-first CLI for recreating a chosen developer environment on Windows.`
  - Persistent Flags:
    - `--no-color` (bool, default `false`, description `Disable color output`)
    - `--verbose` / `-v` (bool, default `false`, description `Enable verbose diagnostic logging`)
  - Subcommands attached in `init()`: `newVersionCmd()`.
  - Signature:
    ```go
    package cli

    import "github.com/spf13/cobra"

    func NewRootCmd() *cobra.Command
    func Execute() error
    ```
- **Error handling**: `Execute()` returns `err` to `main.go`. `SilenceUsage: true` on root so argument/runtime errors do not dump full usage text inappropriately.

### 4. Version Subcommand
- **Target**: `internal/cli/version.go`
- **Action**: Define `version` subcommand.
- **Specifications**:
  - Command: `version`
  - Short: `Print HelloKit version and platform information`
  - Flags:
    - `--json` (bool, default `false`, description `Output version information in JSON format`)
  - Behavior:
    - When `--json` is false: prints `version.Get().String()`.
    - When `--json` is true: serializes `version.Get()` with `json.MarshalIndent` and prints to stdout.

### 5. Executable Entry Point
- **Target**: `cmd/hellokit/main.go`
- **Action**: Implement main entry point forwarding exit status.
- **Implementation**:
  ```go
  package main

  import (
      "os"

      "hellokit/internal/cli"
  )

  func main() {
      if err := cli.Execute(); err != nil {
          os.Exit(1)
      }
  }
  ```

### 6. CLI Unit and Command Tests
- **Target**: `internal/cli/root_test.go`, `internal/cli/version_test.go`
- **Action**: Write isolated tests utilizing `cobra.Command.SetOut` and `cobra.Command.SetErr` with `bytes.Buffer`.
- **Test Cases**:
  - `TestRootCmd_Help`: Run `NewRootCmd()` with args `[]string{"--help"}`. Assert output contains `One command. Your dev environment.` and exit error is `nil`.
  - `TestRootCmd_UnknownFlag`: Run with `[]string{"--bogus"}`. Assert error returned.
  - `TestVersionCmd_Text`: Run `NewRootCmd()` with args `[]string{"version"}`. Assert output contains `0.1.0-dev` and `windows` (or current OS).
  - `TestVersionCmd_JSON`: Run `NewRootCmd()` with args `[]string{"version", "--json"}`. Assert valid JSON containing `"version": "0.1.0-dev"`.

### 7. Documentation & Implementation Plan Tracking
- **Target**: `README.md` and `docs/IMPLEMENTATION_PLAN.md`
- **Action**:
  - Add verified build and test commands to `README.md` under a `## Development & Build` section:
    ```powershell
    go test ./...
    go build -o hellokit.exe ./cmd/hellokit
    .\hellokit.exe --help
    .\hellokit.exe version
    ```
  - Mark checked items in `docs/IMPLEMENTATION_PLAN.md` under `## M0 — Repository and specifications` upon verification.

### 8. GitHub Actions CI Configuration
- **Target**: `.github/workflows/ci.yml`
- **Action**: Add automated Windows workflow running on PR and main push:
  - Runner: `windows-latest`
  - Steps:
    - `actions/checkout@v4`
    - `actions/setup-go@v5` with Go 1.24
    - `go fmt ./...` check (fails if unformatted)
    - `go vet ./...`
    - `go test -v ./...`
    - `go build -v ./cmd/hellokit`

## Critical Files & Anchors
- `cmd/hellokit/main.go`: Main executable entry point calling `cli.Execute()`.
- `internal/cli/root.go`: Root command definition, flags, and command wiring.
- `internal/cli/version.go`: Version command implementation with text and JSON formats.
- `internal/version/version.go`: Build-time variables (`Version`, `Commit`, `Date`) and platform metadata.
- `docs/IMPLEMENTATION_PLAN.md`: M0 checklist and gate tracking.

## Verification
1. **Module & Dependency Resolution**:
   - `go mod tidy` completes with zero errors and generates valid `go.sum`.
2. **Code Quality**:
   - `go fmt ./...` produces no diffs.
   - `go vet ./...` passes cleanly without warnings.
3. **Unit Tests**:
   - `go test -v ./...` passes all tests in `internal/cli` and `internal/version`.
4. **Binary Build & Execution**:
   - `go build -o hellokit.exe ./cmd/hellokit` builds executable.
   - Run `.\hellokit.exe --help`:
     - Exit code: 0
     - Output contains `One command. Your dev environment.` and `Usage: hellokit [command]`.
   - Run `.\hellokit.exe version`:
     - Exit code: 0
     - Output contains `hellokit 0.1.0-dev (windows/amd64)`.
   - Run `.\hellokit.exe version --json`:
     - Exit code: 0
     - Output is valid JSON containing `"version": "0.1.0-dev"`.

## Assumptions & Contingencies
- **Module Path**: Assumes `module hellokit` as specified in `AGENTS.md` ("do not invent a final repository URL").
  - *Fallback*: If a fully-qualified domain path is requested in the future (e.g. `github.com/hellokit/hellokit`), a single `go mod edit -module ...` command and import find/replace updates all packages cleanly.
- **Go Version Directive**: Targets Go `1.24` in `go.mod` to ensure backward compatibility across modern Go toolchains while running on the host's `1.27.1`.
- **Platform Independence of M0**: Root and version commands are pure Go stdlib + Cobra, running cleanly on any developer platform, with tests checking the active platform dynamically.
