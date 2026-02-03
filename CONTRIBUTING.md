# Contributing to VerustCode

Thank you for your interest in contributing to VerustCode! This document provides guidelines and instructions for contributing to the project.

## Development Environment Setup

### Prerequisites

- **Go**: 1.21 or higher ([Download](https://go.dev/dl/))
- **Node.js**: 18.x or higher ([Download](https://nodejs.org/))
- **Git**: Latest version
- **Make**: For running build commands (usually pre-installed on Unix systems)

### Initial Setup

1. **Fork and clone the repository**:
   ```bash
   git clone https://github.com/your-username/verustcode.git
   cd verustcode
   ```

2. **Install Go dependencies**:
   ```bash
   go mod download
   ```

3. **Install frontend dependencies**:
   ```bash
   cd frontend
   npm install
   cd ..
   ```

4. **Build the project**:
   ```bash
   make build
   ```

5. **Run tests**:
   ```bash
   make test
   ```

6. **Start development server**:
   ```bash
   make dev
   ```

The server will start at `http://localhost:8091`. Access the admin dashboard at `http://localhost:8091/admin`.

### Configuration

1. Copy the bootstrap configuration example:
   ```bash
   cp config/bootstrap.example.yaml config/bootstrap.yaml
   ```

2. Edit `config/bootstrap.yaml` as needed (or use environment variables).

3. Configure runtime settings via the admin web interface after first launch.

## Code Style Guidelines

Please follow the project's coding standards defined in [`.cursor/rules/project.mdc`](.cursor/rules/project.mdc). Key points:

### Language Policy

- **Code/comments/docs/commits**: English only
- **User communication**: Chinese

### Code Quality

- Follow SOLID principles
- No magic values: use constants or config
- Never ignore errors: wrap with context (`fmt.Errorf("context: %w", err)`)
- Structured logging with diagnostics
- Validate inputs early; use guard clauses
- Never panic in library code; recover at boundaries

### Error Handling

- Define typed errors in `pkg/errors/`
- Use `errors.Is()` and `errors.As()`
- Lowercase messages, no trailing punctuation
- Always wrap errors with context: `fmt.Errorf("operation failed: %w", err)`

### Code Formatting

- **Go**: Run `make fmt` before committing
- **Frontend**: Follow existing ESLint configuration
- Use `make check` to verify formatting and run `go vet`

### Testing

- Write tests for new features and bug fixes
- Run `make test` to execute all tests
- Run `make test-coverage` to check test coverage
- Aim for >60% test coverage for new code
- Use table-driven tests where appropriate

## Commit Message Format

We follow a conventional commit message format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

### Examples

```
feat(api): add webhook retry mechanism

Implement automatic retry for failed webhook deliveries with
exponential backoff. Add configuration options for max retries
and backoff duration.

Closes #123
```

```
fix(engine): handle nil pointer in review runner

Check for nil review before accessing fields to prevent panic
when review creation fails.

Fixes #456
```

## Pull Request Process

### Before Submitting

1. **Update your fork**: Sync with upstream repository
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**: Follow code style guidelines

4. **Run tests and checks**:
   ```bash
   make test
   make check
   make test-coverage  # Check coverage
   ```

5. **Commit your changes**: Use conventional commit format

6. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

### PR Checklist

- [ ] Code follows project style guidelines
- [ ] Tests added/updated and passing
- [ ] Documentation updated (if needed)
- [ ] Commit messages follow conventional format
- [ ] No merge conflicts with main branch
- [ ] All CI checks passing

### PR Description Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
Describe how you tested your changes

## Related Issues
Closes #issue-number
```

## Testing Requirements

### Unit Tests

- Write unit tests for all new functions/methods
- Test both success and error cases
- Use table-driven tests when appropriate

### Integration Tests

- Test API endpoints with real handlers
- Test database operations with test database
- Clean up test data after tests

### Running Tests

```bash
# Run all tests
make test

# Run tests without race detector (faster)
make test-fast

# Run tests with coverage
make test-coverage

# Run specific test package
go test ./internal/api/...
```

## Project Structure

```
verustcode/
├── cmd/verustcode/     # Main application entry point
├── internal/           # Internal packages (not importable)
│   ├── api/          # HTTP API handlers and routes
│   ├── engine/        # Review/report execution engine
│   ├── dsl/           # DSL parser and validator
│   ├── llm/           # LLM client implementations
│   ├── git/           # Git provider integrations
│   └── ...
├── pkg/               # Public packages (importable)
│   ├── errors/        # Error definitions
│   ├── logger/        # Logging utilities
│   └── ...
├── frontend/          # React frontend application
├── config/            # Configuration examples
└── docs/              # Documentation
```

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/verustcode/verustcode/issues)
- **Discussions**: [GitHub Discussions](https://github.com/verustcode/verustcode/discussions)

## Code Review Process

1. All PRs require at least one approval
2. Maintainers will review code for:
   - Code quality and style
   - Test coverage
   - Documentation completeness
   - Security considerations
3. Address review comments promptly
4. Squash commits if requested before merge

## Questions?

If you have questions about contributing, feel free to:
- Open an issue with the `question` label
- Start a discussion in GitHub Discussions

Thank you for contributing to VerustCode! 🎉
