# collapser-grpc

Collapser is a gRPC sidecar that prevents thundering-herd effects by collapsing identical in-flight requests and fanning out a single backend response.

## Overview

Collapser acts as a request deduplication layer in front of your backend services. When multiple identical requests arrive simultaneously, Collapser executes the backend call only once and broadcasts the response to all waiting clients, significantly reducing load on downstream services.

## Features

- **Request Collapsing**: Automatically deduplicates identical in-flight requests
- **gRPC Native**: Built with gRPC for high performance
- **Sidecar Pattern**: Deploy alongside your services without code changes
- **Thundering Herd Prevention**: Protects backends from request spikes

## Quick Start

### Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (protoc) - optional, for generating proto code

### Installation

```bash
git clone https://github.com/VarunGitGood/collapser-grpc.git
cd collapser-grpc
make deps
make build
```

### Running

```bash
make run
```

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Available Make Targets

- `make build` - Build the binary
- `make test` - Run tests
- `make run` - Build and run the application
- `make clean` - Clean build artifacts
- `make fmt` - Format code
- `make vet` - Run go vet
- `make lint` - Run linter
- `make proto` - Generate code from proto files

## Project Structure

```
.
├── cmd/collapser/           # Main application entry point
├── pkg/collapser/           # Public library code
├── internal/collapser/      # Private implementation
├── api/proto/               # gRPC protocol definitions
└── Makefile                 # Build automation
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to contribute to this project.

## License

[Add your license here]
