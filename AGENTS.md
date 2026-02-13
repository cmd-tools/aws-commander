# AGENTS.md

Guide for AI coding agents working in the `aws-commander` repository.

## Project Overview

Go TUI application wrapping the AWS CLI, built with `tview`/`tcell`. Uses YAML
configs in `configurations/` to define AWS services and commands. The UI is
widget-based with global state in `cmd/uiState.go` and keyboard shortcuts
dispatched via `App.SetInputCapture`.

Go version: **1.24.0** (see `go.mod`).

## Build / Test / Run

```bash
go build ./...                                        # Build
go test ./...                                         # Run all tests
go test ./parser/ -run Test_ParseCommand_Object       # Single test by name
go test ./ui/ -run TestConvertDynamoDBToRegularJSON   # Single test by name
go test ./parser/ -v                                  # All tests in a package
go test -race ./...                                   # Race detector
go vet ./...                                          # Static analysis
gofmt -l .                                            # Format check
go mod tidy                                           # Tidy modules
```

There is no linter config (`.golangci.yml`). There is no CI pipeline. The
`Makefile` is only for LocalStack (Docker-based local AWS): `make up` / `make down`.

## Package Structure

| Package | Purpose |
|---------|---------|
| `main` (root) | Entry point, handlers, views, navigation, search, DynamoDB UI |
| `cmd/` | Domain types (`Command`, `Resource`, `Action`), config loading, `UIState` |
| `cmd/profile/` | AWS profile discovery and SSO handling |
| `parser/` | Command output parsing AND content view creation |
| `ui/` | Reusable TUI components (table, list, tree, modal, toast, shortcuts) |
| `executor/` | Shell command execution wrapper |
| `logger/` | `zerolog` logger singleton |
| `constants/` | Shared string constants |
| `helpers/` | String utils, AWS version detection |

Dependency direction: `main` -> `parser`, `ui`, `cmd` -> `executor` -> `logger`.
`constants` and `logger` are leaf packages. No circular dependencies.

## Code Style

### Formatting

Use `gofmt`. No custom formatter config exists.

### Imports

Two groups separated by a blank line: standard library first, then all
external and internal packages together (alphabetized within each group).

```go
import (
    "fmt"
    "strings"

    "github.com/cmd-tools/aws-commander/cmd"
    "github.com/cmd-tools/aws-commander/logger"
    commandParser "github.com/cmd-tools/aws-commander/parser"
    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
)
```

Known alias: `commandParser` for the `parser` package in `handlers.go`.
Side-effect imports use `_` (e.g., `_ "image/png"` for codec registration).

### Naming

- **Exported**: `PascalCase` -- `CreateContentView`, `ParseCommand`, `UIState`
- **Unexported**: `camelCase` -- `handleEscKey`, `buildContentTree`
- **Files**: mostly camelCase (`customTable.go`, `uiState.go`); some snake_case
  (`string_helper.go`). Follow existing file names in each directory.
- **Constants**: `PascalCase` for exported (`EmptyString`, `ToastRefreshMessage`)
- **Constructors**: `Create*` prefix (`CreateCustomTableView`, `CreateInputForm`)
- **Config structs**: `*Properties` suffix (`CustomTableViewProperties`, `InputFormProperties`)
- **Handlers**: `handle*` prefix (`handleEscKey`, `handleNextPage`)
- **Test functions**: `Test_PascalCase_SubCase` (older) or `TestPascalCase` (newer)

### Types

- Structs use YAML struct tags: `` `yaml:"fieldName"` ``
- Pointer receivers for mutation, value receivers for read-only
- One interface exists (`Boxed` in `ui/toast.go`) -- keep interfaces small
- Custom string types for enums: `type BreadcrumbType string` with `const` block
- No `iota` usage; constants are explicit string values

### Error Handling

Errors are logged and handled locally -- never wrapped with `%w`, never
propagated up the call stack. Common patterns:

```go
// Pattern 1: Log and return fallback
if err != nil {
    logger.Logger.Error().Err(err).Msg("Failed to parse JSON")
    return createTextContentView(commandName, fileName, content)
}

// Pattern 2: Log and continue
if err != nil {
    logger.Logger.Error().Msg(fmt.Sprintf("Error: %v", err))
}
```

Use `logger.Logger` (zerolog) for all logging. Prefer structured fields
(`.Str()`, `.Int()`, `.Err()`) over `fmt.Sprintf` in the `.Msg()` call:

```go
// Preferred
logger.Logger.Debug().Str("key", value).Msg("Description")

// Acceptable (older style)
logger.Logger.Debug().Msg(fmt.Sprintf("Description: %s", value))
```

### Comments

- `//` doc comments above exported functions (required)
- Inline `//` comments for non-obvious logic
- Trailing `//` comments on struct fields for documentation

### Global State

The app uses package-level globals extensively:
- `cmd.UiState` -- singleton UI state
- `cmd.Resources` -- loaded AWS service configs
- `main.App`, `main.Body`, `main.Search` -- tview application globals

When adding new state, add fields to `cmd.UIState` in `cmd/uiState.go`.
Always clean up state fields when navigating away (see `handleDependentCommandBack`).

### Shortcuts

`defaultKeyCombinations()` in `handlers.go` returns all active shortcuts.
The list is rebuilt on every `updateRootView` call, so conditional shortcuts
appear/disappear based on `cmd.UiState` flags. Use `else if` branches for
mutually exclusive shortcuts (e.g., `v` for DynamoDB vs content view toggle).
Return `nil` to consume an event, `event` to pass through.

### YAML Configs

Service definitions live in `configurations/*.yaml`. Each defines a `Resource`
with `commands`. Commands can have `depends_on`, `actions`, `pagination`, and
`parse` directives. The `view` field determines rendering (`tableView`,
`contentView`).

## Testing

Only stdlib `testing` is used (no testify/gomega). Two test files exist:
- `parser/command_test.go` -- basic parse tests
- `ui/customJsonViewer_test.go` -- table-driven tests with `t.Run()`

Prefer table-driven tests with `t.Run()` subtests for new tests:

```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {name: "case 1", input: "...", expected: "..."},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FunctionUnderTest(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```