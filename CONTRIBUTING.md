# Contributing to Aetheris

Thank you for your interest in contributing to Aetheris!

## How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b my-feature`)
3. Commit your changes with clear messages
4. Push to your fork (`git push origin my-feature`)
5. Open a Pull Request

## Development Setup

### Prerequisites

- **Go 1.25.7+** (see [go.mod](../go.mod) for exact version)
- **Docker** (for running PostgreSQL and other services)
- **Make** (optional, for convenience commands)

### Quick Start

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/CoRag.git
cd CoRag

# Install dependencies
go mod download

# Build all binaries
make build

# Run tests
make test

# Start local stack (Postgres + API + Workers)
make docker-run
```

## Code Style

- Use `gofmt` for Go code formatting
- Run `golangci-lint` before submitting PRs
- Maintain existing module and package structure

### Git hooks

To run `gofmt` automatically on staged Go files before each commit, enable the project hooks (local to this repo only):

```bash
git config core.hooksPath .githooks
```

Or run the install script once: `./scripts/install-hooks.sh`

## Testing

### Running Tests

```bash
# Run all tests
go test -v -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out -covermode=atomic ./...

# Run specific package tests
go test -v ./internal/agent/...
```

### Writing Tests

- All core packages should have unit tests
- Follow the existing test patterns in the project
- Use table-driven tests where appropriate
- **Coverage Requirements**:
  - Overall code coverage must be ≥50% (checked in CI)
  - Core modules (internal/runtime, internal/agent, internal/pipeline) should aim for ≥60%
  - Run `go test -coverprofile=coverage.out -covermode=atomic ./...` to check coverage
  - Review coverage report with `go tool cover -func=coverage.out`

### Test Naming and Organization

```bash
# Test files should be named *_test.go
# Place tests in the same package as the code being tested
package mypackage

# Use table-driven tests for multiple test cases
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "input1", "expected1"},
        {"case 2", "input2", "expected2"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FunctionName(tt.input)
            if result != tt.expected {
                t.Errorf("expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

### Integration Tests

```bash
# Run integration tests (requires Postgres)
make test-integration

# Or manually:
docker start aetheris-pg
psql "$TEST_JOBSTORE_DSN" -f internal/runtime/jobstore/schema.sql
go test -v ./internal/runtime/jobstore ./internal/agent/job
```

## Documentation

We welcome documentation contributions! Here's how to help:

### Documentation Structure

```
docs/
├── README.md          # Entry point
├── guides/            # User guides and tutorials
│   └── tutorials/     # Recipe-style tutorials
├── concepts/          # Concept explanations
├── reference/         # API/config reference
└── blog/              # Technical articles
```

### Writing Tutorials

We follow a recipe-style format for tutorials. See existing examples:

- [docs/guides/tutorials/code-review-agent.md](docs/guides/tutorials/code-review-agent.md)
- [docs/guides/tutorials/audit-agent.md](docs/guides/tutorials/audit-agent.md)
- [docs/guides/tutorials/long-running-tasks.md](docs/guides/tutorials/long-running-tasks.md)

Key requirements:
1. Use Markdown format
2. Include runnable code examples
3. Explain concepts step-by-step
4. Provide expected output

## CI Requirements

All PRs must pass the CI checks:

1. **Build** - Code compiles successfully
2. **Vet** - No `go vet` warnings
3. **Format** - Code is formatted with `gofmt`
4. **Lint** - Pass golangci-lint checks
5. **Tests** - All tests pass with race detector (`-race`)
6. **Coverage** - Overall code coverage ≥50% (checked in CI)

You can run these locally:

```bash
make build
make vet
make fmt-check
make test
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
```

### Coverage Thresholds

| Module Type | Minimum Coverage |
|-------------|------------------|
| Overall     | ≥50%             |
| Core (internal/runtime, internal/agent, internal/pipeline) | ≥60% |

### Running Specific Test Suites

```bash
# Run only short tests (skip integration tests)
go test -short -v ./...

# Run tests for specific package
go test -v ./pkg/config/...
go test -v ./internal/runtime/...

# Run tests with coverage for specific package
go test -coverprofile=coverage.out -covermode=atomic ./pkg/config/...
```

## Reporting Issues

- Check existing issues before creating a new one
- Provide steps to reproduce, expected behavior, and screenshots if applicable
- For bugs, include:
  - Go version
  - Operating system
  - Steps to reproduce
  - Expected vs actual behavior

## Pull Request Guidelines

- Keep PRs focused and reasonably sized
- Include tests for new features
- Update documentation if needed
- Follow the commit message format:
  - `feat: add new feature`
  - `fix: resolve issue #123`
  - `docs: update tutorial`
  - `test: add tests for module`

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.

## Getting Help

- Check the [documentation](../docs/)
- Join our community discussions
- Open an issue for bugs or feature requests
