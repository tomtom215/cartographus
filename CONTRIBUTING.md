# Contributing to Cartographus

**Last Verified**: 2026-01-11

Thank you for your interest in contributing to Cartographus! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Standards](#code-standards)
- [Documentation](#documentation)
- [Community](#community)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.24+** (required for backend)
- **Node.js 18+** and **npm** (required for frontend)
- **Docker and Docker Compose** (recommended for testing)
- **Git** (for version control)
- **CGO-enabled environment** (DuckDB requires CGO)

### Understanding the Codebase

- Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for system design and architecture decisions
- Review [CLAUDE.md](CLAUDE.md) for comprehensive development guidelines
- Check [README.md](README.md) for feature overview and API documentation
- Review [docs/TESTING.md](docs/TESTING.md) for testing strategy and E2E test coverage

## Development Setup

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/cartographus.git
cd cartographus

# Add upstream remote
git remote add upstream https://github.com/tomtom215/cartographus.git
```

### 2. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install frontend dependencies
cd web
npm install
cd ..
```

### 3. Configure Environment

```bash
# Copy environment template
cp .env.example .env

# Edit .env with your configuration
# At minimum, set:
# - TAUTULLI_URL=http://your-tautulli:8181
# - TAUTULLI_API_KEY=your_api_key
# - AUTH_MODE=none (for development)
```

### 4. Build and Run

```bash
# Build frontend
cd web
npm run build
cd ..

# Run backend
CGO_ENABLED=1 go run cmd/server/main.go

# Or use Docker Compose
docker-compose up -d
```

### 5. Verify Setup

Open http://localhost:3857 in your browser and verify the application loads correctly.

## Making Changes

### Branch Strategy

- **main**: Production-ready code
- **feature/***: New features (e.g., `feature/add-user-analytics`)
- **fix/***: Bug fixes (e.g., `fix/resolve-sync-error`)
- **docs/***: Documentation changes (e.g., `docs/update-api-reference`)

### Creating a Feature Branch

```bash
# Fetch latest upstream changes
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name
```

### Development Workflow

1. **Make your changes** in your feature branch
2. **Follow code standards** (see below)
3. **Add tests** for new functionality
4. **Update documentation** as needed
5. **Run tests locally** before committing
6. **Commit with conventional commits** (see below)

## Testing

### Running Tests

```bash
# Backend unit tests (MUST include build tags)
go test -tags "wal,nats" -v -race ./...

# Backend tests with coverage
go test -tags "wal,nats" -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Backend benchmarks
go test -tags "wal,nats" -bench=. -benchmem ./internal/database

# Frontend type checking
cd web
npx tsc --noEmit

# End-to-end tests (requires running server)
cd web
npm run test:e2e

# All tests
make test-all
```

### Test Coverage Requirements

- **Minimum coverage**: 78% per package
- **New features**: Must include unit tests
- **Bug fixes**: Must include regression tests
- **Critical paths**: Aim for 90%+ coverage

### Writing Tests

- Use **table-driven tests** for Go code
- Follow existing test patterns in `*_test.go` files
- Mock external dependencies (database, API clients)
- Test both happy paths and error cases

Example:

```go
func TestHandler_Stats(t *testing.T) {
    tests := []struct {
        name       string
        setupMock  func(*MockDB)
        wantStatus int
        wantErr    bool
    }{
        {
            name: "successful stats retrieval",
            setupMock: func(m *MockDB) {
                m.GetStatsFunc = func() (Stats, error) {
                    return Stats{TotalPlaybacks: 100}, nil
                }
            },
            wantStatus: 200,
            wantErr:    false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Submitting Changes

### Before Submitting

Run the **MANDATORY Pre-Commit Checklist** (see CLAUDE.md):

```bash
# 1. Format and cleanup
gofmt -s -w .
go mod tidy
cd web && npm run build && cd ..

# 2. Lint and verify (MUST include build tags)
go vet -tags "wal,nats" ./...
cd web && npx tsc --noEmit && cd ..

# 3. Test (MUST include build tags)
go test -tags "wal,nats" -v -race ./...

# 4. Build (MUST include build tags)
CGO_ENABLED=1 go build -tags "wal,nats" -o cartographus ./cmd/server
```

### Commit Message Format

Use **Conventional Commits** format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions/modifications
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build/tooling changes
- `ci`: CI/CD changes

**Examples:**

```bash
git commit -m "feat(api): add concurrent streams analytics endpoint"
git commit -m "fix(auth): resolve JWT token expiration issue"
git commit -m "docs: update API reference with new endpoints"
git commit -m "test: add unit tests for filter builder"
```

### Opening a Pull Request

1. **Push your branch** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Open a Pull Request** on GitHub:
   - Go to https://github.com/tomtom215/cartographus/compare
   - Select your fork and branch
   - Fill out the PR template completely

3. **PR Title**: Use conventional commit format
   ```
   feat: Add bandwidth usage analytics dashboard
   ```

4. **PR Description**: Include:
   - **Summary**: What does this PR do?
   - **Changes**: Bullet list of changes
   - **Testing**: How was it tested?
   - **Documentation**: What docs were updated?
   - **Screenshots**: For UI changes

5. **Wait for review**: Maintainers will review and provide feedback

### PR Review Process

- **Automated checks** must pass (lint, tests, build)
- **Code review** from at least one maintainer required
- **Changes requested**: Address feedback and push updates
- **Approval**: Once approved, maintainer will merge

## Code Standards

### Go Code Standards

- **Format**: Use `gofmt -s` for formatting
- **Linting**: Code must pass `go vet ./...`
- **Error handling**: Always handle errors explicitly
- **Naming**: Follow Go naming conventions
  - Exported functions: `PascalCase`
  - Unexported functions: `camelCase`
  - Constants: `PascalCase`
- **Comments**: Public APIs must have godoc comments
- **No CGO disable**: DuckDB requires `CGO_ENABLED=1`

### TypeScript Code Standards

- **Strict mode**: All code must compile with TypeScript strict mode
- **No implicit any**: All types must be explicit
- **Format**: Follow existing code style
- **Null checks**: Use optional chaining (`?.`) and nullish coalescing (`??`)
- **ES modules**: Use modern ES module syntax

### Database Standards

- **DuckDB syntax**: Not SQLite! Use DuckDB-specific functions
- **Parameterized queries**: Always use `?` placeholders
- **Indexes**: Add indexes for filtered columns
- **Spatial queries**: Use spatial extension functions (`ST_*`)

### Security Standards

- **Input validation**: Validate all user inputs
- **SQL injection**: Use parameterized queries
- **XSS prevention**: Escape HTML output
- **Secrets**: Never commit secrets or API keys
- **Authentication**: Respect existing auth modes

## Documentation

### When to Update Documentation

Update documentation when you:
- Add new features or endpoints
- Change existing behavior
- Add configuration options
- Modify database schema
- Update dependencies

### Which Files to Update

- **README.md**: User-facing features, API endpoints, configuration
- **ARCHITECTURE.md**: System design, architecture decisions
- **CLAUDE.md**: Development guidelines, codebase structure
- **CHANGELOG.md**: All user-visible changes (required for every PR)
- **Code comments**: Inline documentation for complex logic

### Documentation Standards

- **Accuracy**: Verify all claims against source code
- **Examples**: Provide working code examples
- **Completeness**: Document all parameters and options
- **Consistency**: Match terminology across all docs
- **No emojis**: Maintain professional tone

## Community

### Getting Help

- **GitHub Issues**: https://github.com/tomtom215/cartographus/issues
- **GitHub Discussions**: https://github.com/tomtom215/cartographus/discussions
- **Questions**: Tag your issue with `question` label

### Reporting Bugs

When reporting bugs, include:

1. **Environment**: OS, Go version, Docker version
2. **Configuration**: Relevant environment variables (redact secrets)
3. **Steps to reproduce**: Minimal reproduction steps
4. **Expected behavior**: What should happen
5. **Actual behavior**: What actually happens
6. **Logs**: Relevant log output (set `LOG_LEVEL=debug`)
7. **Screenshots**: For UI bugs

### Suggesting Features

When suggesting features, include:

1. **Use case**: Why is this feature needed?
2. **Current workaround**: How do you handle this now?
3. **Proposed solution**: How should it work?
4. **Alternatives**: Other approaches considered?
5. **Impact**: Who benefits from this feature?

### Project Structure

```
cartographus/
├── cmd/server/          # Application entry point
├── internal/            # Private application code (26 packages)
│   ├── api/            # HTTP handlers and Chi routing (302 endpoints)
│   ├── auth/           # Authentication (JWT, OIDC, Plex)
│   ├── authz/          # Zero Trust authorization (Casbin)
│   ├── cache/          # In-memory caching (LFU)
│   ├── config/         # Configuration management (Koanf v2)
│   ├── database/       # DuckDB abstraction (62 source files)
│   ├── detection/      # Security detection engine (5 rules)
│   ├── eventprocessor/ # NATS/Watermill event processing
│   ├── logging/        # Structured logging (zerolog)
│   ├── supervisor/     # Suture v4 process supervision
│   ├── sync/           # Multi-server sync (Plex, Tautulli)
│   ├── wal/            # BadgerDB Write-Ahead Log
│   ├── websocket/      # Real-time WebSocket hub
│   └── ...             # Additional packages (see docs/ARCHITECTURE.md)
├── web/                # Frontend application (229 TypeScript files)
│   ├── src/            # TypeScript source
│   ├── public/         # Static assets
│   └── dist/           # Build output (gitignored)
├── tests/e2e/          # Playwright E2E tests (75 suites, 1300+ tests)
└── .github/workflows/  # CI/CD pipelines
```

## License

By contributing to Cartographus, you agree that your contributions will be licensed under the [AGPL-3.0 License](LICENSE).

---

Thank you for contributing to Cartographus! Your contributions help make this project better for everyone.
