# Contributing to Collapser

Thank you for your interest in contributing to Collapser!

## Development Setup

1. Install Go 1.21 or later
2. Clone the repository:
   ```bash
   git clone https://github.com/VarunGitGood/collapser-grpc.git
   cd collapser-grpc
   ```

3. Install dependencies:
   ```bash
   make deps
   ```

## Building

```bash
make build
```

## Running Tests

```bash
make test
```

## Code Style

- Run `make fmt` to format your code
- Run `make vet` to check for common issues
- Run `make lint` to run the linter (requires golangci-lint)

## Project Structure

```
.
├── cmd/           # Main applications
├── pkg/           # Public library code
├── internal/      # Private application code
├── api/           # Protocol definitions (proto files)
└── Makefile       # Build and test commands
```

## Pull Request Process

1. Fork the repository
2. Create your feature branch
3. Make your changes
4. Write or update tests
5. Ensure all tests pass
6. Submit a pull request
