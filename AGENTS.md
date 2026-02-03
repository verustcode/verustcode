# VerustCode Development Guide

DSL-driven AI code review tool with multi-reviewer pipeline architecture.

## Tech Stack

- Go 1.21+
- YAML configuration
- SQLite (`modernc.org/sqlite`)
- Pluggable CLI agents

## Language Policy

- **Code, comments, docs, commits**: English only
- **User communication**: Chinese

## Code Quality Standards

### SOLID Principles

Follow SOLID design principles for maintainable, extensible code.

### No Magic Values

- Use constants for fixed values
- Use config files for environment-specific values
- Define constants in `constants.go` per package

### Error Handling

- Never ignore errors
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Define typed errors in `pkg/errors/`
- Use `errors.Is()` and `errors.As()` for error checking
- Error messages: lowercase, no trailing punctuation

### Input Validation

- Validate inputs early
- Use guard clauses for readability
- Fail fast on invalid input

### Logging

- Use structured logging with diagnostics
- Include relevant context in log messages

### Panic Recovery

- Never panic in library code
- Recover at boundaries (main, HTTP handlers)

## Security Requirements

- **Default deny**: All API endpoints require authorization by default
- **Authentication middleware**: Apply to all routes unless explicitly documented as public
- **Least privilege**: Deny by default, allow by exception

## Code Style

### Go

- Format with `gofmt`
- Lint with `golint`
- Interface-driven design
- Prefer composition over inheritance

### DSL (YAML)

- Clear hierarchy
- Use `rule_base` for shared configurations
- Support inheritance and overrides

## DSL Design Principles

- **Declarative**: Define "what" reviewers should focus on, not "how" they execute
- **Composable**: Multiple reviewers can be combined and run sequentially
- **Configurable**: Support inheritance, overrides, and environment variables
- **Extensible**: New reviewers, agents, and output channels can be added easily
