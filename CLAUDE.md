# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is xk6-encoding, a k6 extension that provides JavaScript's TextEncoder and TextDecoder APIs for handling various text encodings in k6 performance tests. The extension supports UTF-8, UTF-16LE, UTF-16BE, and includes proper BOM handling.

- This k6 extensions aims to implement the javascript/WebAPI Encoding living standard: https://encoding.spec.whatwg.org/#interface-textencoder, but only cares about a limited subset of encodings: utf-8, and utf-16.

## Development Commands

### Building the Extension
```bash
# Build a custom k6 binary with the extension
xk6 build --with github.com/oleiade/xk6-encoding@latest
```

### Running Tests
```bash
# Run all Go tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test files
go test -v ./encoding/

# Run a single test function
go test -v -run TestTextDecoder ./encoding/
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run linter (if golangci-lint is available)
golangci-lint run

# Tidy dependencies
go mod tidy
```

## Architecture

### Core Components

**Module Structure (`encoding/module.go`)**
- `RootModule`: Main extension entry point implementing k6's module interface
- `ModuleInstance`: Per-VU instance containing TextEncoder/TextDecoder
- JS constructor functions that bridge Go implementations to JavaScript objects

**Text Processing (`encoding/text_decoder.go`, `encoding/text_encoder.go`)**
- `TextDecoder`: Handles decoding byte streams to strings with encoding support
- `TextEncoder`: Handles encoding strings to UTF-8 byte streams
- Uses `golang.org/x/text/encoding` for charset handling

**Testing Infrastructure (`encoding/test_setup.go`)**
- Web Platform Tests (WPT) compatibility layer
- Goja runtime setup with k6 module integration
- JavaScript test harness for running encoding tests

### Key Design Patterns

1. **k6 Module Pattern**: Implements k6's module interface with `RootModule.NewModuleInstance()`
2. **JavaScript Bridge**: Uses Goja runtime reflection to expose Go methods as JS functions
3. **Streaming Support**: TextDecoder supports chunked decoding with internal buffering
4. **Error Handling**: Custom error types (RangeError, TypeError) that match Web API standards

### Testing Strategy

- **Unit Tests**: Go tests for core encoding/decoding functionality
- **Integration Tests**: JavaScript tests that run via Goja runtime
- **WPT Compatibility**: Tests based on Web Platform Test specifications
- **Test Scripts**: Located in `encoding/tests/` directory with utility functions

### Supported Encodings

The extension supports these text encodings:
- UTF-8 (default)
- UTF-16LE (little endian)
- UTF-16BE (big endian)
- BOM handling (configurable via `ignoreBOM` option)

## File Structure

```
encoding/
├── module.go          # k6 module interface and JS constructors
├── text_decoder.go    # TextDecoder implementation
├── text_encoder.go    # TextEncoder implementation
├── error.go          # Custom error types
├── goja.go           # Goja runtime utilities
├── test_setup.go     # Test infrastructure
├── *_test.go         # Go unit tests
└── tests/            # JavaScript test files
```