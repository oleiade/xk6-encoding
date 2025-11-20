# xk6-encoding

A [k6](https://go.k6.io/k6) extension that provides JavaScript's TextEncoder and TextDecoder APIs for handling various text encodings in k6 performance tests. This extension implements a subset of the [Encoding Living Standard](https://encoding.spec.whatwg.org/) with focus on UTF-8 and UTF-16 encodings.

## Features

- **TextEncoder**: Encode strings to UTF-8 byte arrays with proper surrogate handling
- **TextDecoder**: Decode byte arrays to strings with support for multiple encodings
- **Streaming support**: Handle large data streams efficiently (with some limitations)
- **BOM handling**: Configurable byte order mark processing
- **Multiple encodings**: UTF-8, UTF-16LE, UTF-16BE support
- **Error handling**: Fatal and non-fatal decoding modes

## Supported Encodings

- **UTF-8** (default) - Full support
- **UTF-16LE** (Little Endian) - Full support
- **UTF-16BE** (Big Endian) - Full support

## Known Limitations

The extension now passes the WPT suites we imported for UTF-8 and UTF-16 (including the previously skipped streaming cases), but there are still a few practical constraints to keep in mind:

### Encoding coverage

- Only UTF-8, UTF-16LE, and UTF-16BE are implemented. Other legacy encodings from the Encoding Living Standard are intentionally out of scope for now.
- `TextEncoder` always emits UTF-8 as per the platform API. Supplying other labels to its constructor has no effect beyond surfacing the canonical name in `.encoding`.

### API surface

- The streaming helper interfaces (`TextDecoderStream`, `TransformStream`, etc.) are not exposed. Use repeated `decode()` calls with `{ stream: true }` instead.
- `SharedArrayBuffer` inputs are supported inside the embedded WPT harness we ship, but k6 itself currently only exposes `ArrayBuffer`/TypedArray in regular scripts.

### Operational notes

- Streaming relies on the Go `golang.org/x/text/transform` package under the hood. While our state machine ensures spec-compliant behavior for UTF-8/UTF-16, extremely memory-constrained scenarios may want to reuse decoder instances instead of constructing them per chunk.
- Fatal-mode decoding matches WPT behavior, but be mindful that it raises `TypeError` on the first bad sequence—plan your error handling accordingly.

For most k6 use cases this means you can treat the extension as a drop-in replacement for browser `TextEncoder`/`TextDecoder` when working with UTF-8/UTF-16 data.

## Installation

To build a [k6](https://go.k6.io/k6) binary with this extension, first ensure you have the prerequisites:

- [Go toolchain](https://go101.org/article/go-toolchain.html)
- Git

Then:

1. Install [xk6](https://github.com/grafana/xk6):
```bash
go install go.k6.io/xk6/cmd/xk6@latest
```

2. Build the binary:
```bash
xk6 build --with github.com/oleiade/xk6-encoding@latest
```

## Usage

### TextEncoder

```javascript
import { TextEncoder } from 'k6/x/encoding';

const encoder = new TextEncoder();

// Basic encoding
const encoded = encoder.encode('Hello, World!');
console.log(encoded); // Uint8Array with UTF-8 bytes

// Surrogate handling (works correctly)
const withEmoji = encoder.encode('Hello 🌍 World');
const withSurrogates = encoder.encode('\uD83C\uDF0D'); // 🌍 as surrogate pair

// Empty string handling
const empty = encoder.encode(); // Returns empty Uint8Array
```

### TextDecoder

```javascript
import { TextDecoder } from 'k6/x/encoding';

// Basic usage
const decoder = new TextDecoder();
const bytes = new Uint8Array([72, 101, 108, 108, 111]);
const decoded = decoder.decode(bytes);
console.log(decoded); // "Hello"

// With encoding specification
const utf16Decoder = new TextDecoder('utf-16le');
const utf16Bytes = new Uint8Array([72, 0, 101, 0, 108, 0, 108, 0, 111, 0]);
const utf16Decoded = utf16Decoder.decode(utf16Bytes);
console.log(utf16Decoded); // "Hello"

// Streaming mode (basic scenarios work well)
const streamDecoder = new TextDecoder();
let result = '';
result += streamDecoder.decode(new Uint8Array([72, 101]), {stream: true});
result += streamDecoder.decode(new Uint8Array([108, 108, 111]));
console.log(result); // "Hello"

// Fatal mode
const fatalDecoder = new TextDecoder('utf-8', {fatal: true});
try {
    fatalDecoder.decode(new Uint8Array([0xFF])); // Invalid UTF-8
} catch (e) {
    console.log('Decoding failed:', e.message);
}

// BOM handling
const bomDecoder = new TextDecoder('utf-16le', {ignoreBOM: false});
const withBom = new Uint8Array([0xFF, 0xFE, 72, 0, 105, 0]); // BOM + "Hi"
console.log(bomDecoder.decode(withBom)); // "Hi"
```

### Constructor Options

#### TextDecoder Options

- `label` (string): The encoding label (default: "utf-8")
  - Supported: "utf-8", "utf-16", "utf-16le", "utf-16be", and various aliases
- `options.fatal` (boolean): Throw errors on invalid sequences (default: false)
- `options.ignoreBOM` (boolean): Ignore byte order marks (default: false)

#### Decode Options

- `stream` (boolean): Enable streaming mode for chunked processing (default: false)
  - Note: Complex invalid byte scenarios may have limitations

## Error Handling

The extension provides proper error handling following the Web API specification:

- **RangeError**: Thrown for unsupported encodings
- **TypeError**: Thrown for invalid sequences in fatal mode
- **Replacement characters**: Invalid sequences replaced with U+FFFD in non-fatal mode

## Examples

### File Processing

```javascript
import { TextDecoder } from 'k6/x/encoding';
import { open } from 'k6/experimental/fs';

export default async function() {
    const file = await open('data.txt');
    const decoder = new TextDecoder();
    
    let content = '';
    const buffer = new Uint8Array(1024);
    
    while (true) {
        const bytesRead = await file.read(buffer);
        if (bytesRead === 0) break;
        
        const chunk = buffer.slice(0, bytesRead);
        content += decoder.decode(chunk, {stream: true});
    }
    
    content += decoder.decode(); // Flush remaining data
    console.log('File content:', content);
}
```

### Different Encodings

```javascript
import { TextDecoder, TextEncoder } from 'k6/x/encoding';

export default function() {
    const text = "Hello, 世界! 🌍";
    const encoder = new TextEncoder();
    
    // Encode to UTF-8
    const utf8Bytes = encoder.encode(text);
    
    // Decode with different settings
    const utf8Decoder = new TextDecoder('utf-8');
    const decoded = utf8Decoder.decode(utf8Bytes);
    
    console.log('Original:', text);
    console.log('Decoded:', decoded);
    console.log('Match:', text === decoded);
}
```

### Working with UTF-16

```javascript
import { TextDecoder } from 'k6/x/encoding';

export default function() {
    // UTF-16LE example
    const utf16leDecoder = new TextDecoder('utf-16le');
    const utf16leBytes = new Uint8Array([
        0x48, 0x00,  // 'H'
        0x65, 0x00,  // 'e'
        0x6C, 0x00,  // 'l'
        0x6C, 0x00,  // 'l'
        0x6F, 0x00   // 'o'
    ]);
    console.log(utf16leDecoder.decode(utf16leBytes)); // "Hello"
    
    // UTF-16BE example
    const utf16beDecoder = new TextDecoder('utf-16be');
    const utf16beBytes = new Uint8Array([
        0x00, 0x48,  // 'H'
        0x00, 0x65,  // 'e'
        0x00, 0x6C,  // 'l'
        0x00, 0x6C,  // 'l'
        0x00, 0x6F   // 'o'
    ]);
    console.log(utf16beDecoder.decode(utf16beBytes)); // "Hello"
}
```

## Best Practices

### For Reliable Streaming

```javascript
// ✅ Good: Simple streaming with complete UTF-8 sequences
const decoder = new TextDecoder();
let result = '';
result += decoder.decode(new Uint8Array([0xE2, 0x9C, 0x85]), {stream: true}); // ✅
result += decoder.decode(new Uint8Array([0x20, 0x47, 0x6F, 0x6F, 0x64])); // " Good"

// ⚠️ May have limitations: Complex invalid byte scenarios
// decoder.decode(new Uint8Array([0xF0]), {stream: true});
// decoder.decode(new Uint8Array([0x41])); // May not handle optimally
```

### Error Handling

```javascript
// Handle encoding errors gracefully
function safelyDecode(bytes, encoding = 'utf-8') {
    try {
        const decoder = new TextDecoder(encoding, {fatal: true});
        return decoder.decode(bytes);
    } catch (error) {
        console.warn(`Decoding failed with ${encoding}, falling back to replacement characters`);
        const fallbackDecoder = new TextDecoder(encoding, {fatal: false});
        return fallbackDecoder.decode(bytes);
    }
}
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -v -run TestTextDecoder ./encoding/
```

### Building

```bash
xk6 build --with github.com/oleiade/xk6-encoding@latest
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter (if available)
golangci-lint run

# Tidy dependencies
go mod tidy
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Areas for Contribution

- Enhanced streaming support for edge cases
- Additional encoding support
- Performance optimizations
- Test coverage improvements

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.