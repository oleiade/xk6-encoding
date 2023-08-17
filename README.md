# `TextEncoder` and `TextDecoder` implementations for k6

Welcome to xk6-encoding, an xk6 extension that brings support for Javascript's TextEncoder and TextDecoder to k6, enabling you to seamlessly handle various text encodings during your performance tests.

## Features

* **Text Encoding**: Convert your strings into byte streams with support for various encoding formats including UTF-8, UTF-16, and Windows-1252.
* **Text Decoding**: Decode byte streams back to strings with ease, even when processing the data in chunks.
* **Flexible Options**: Handle Byte Order Marks (BOM) and determine behavior on decoding invalid data.

## Why Use xk6-encoding?

If you're working with systems that utilize various text encodings or if you're aiming to test the performance of encoding/decoding tasks, this extension will be invaluable for your k6 tests.

## Getting Started

1. Make sure you have the latest version of the xk6 tool installed:

```bash
go install go.k6.io/xk6/cmd/xk6@latest
```

2. Build your custom k6 binary:

```bash
xk6 build --with github.com/oleiade/xk6-encoding@latest
```

3. Use in your k6 script:
To encode text:

```javascript
import { TextEncoder } from 'k6/encoding';
const encoder = new TextEncoder("utf-8");
const encoded = encoder.Encode("Your text here");
```

To decode text:
```javascript
import { TextDecoder } from 'k6/encoding';
const decoder = new TextDecoder("utf-8");
const decoded = decoder.Decode(encodedData);
```

4. Run your k6 test with the custom k6 binary you built:

```bash
./k6 run your-test-script.js
```

## Supported Encodings

* **utf-8**: Standard encoding for the web.
* **utf-16le** and **utf-16be**: Unicode encodings that can represent any character in the Unicode standard.
* **windows-1252**: A character encoding of the Latin alphabet, used by default in the legacy components of Microsoft Windows.

## Contributing
Your contributions are always welcome! If you discover an issue or have a feature request, please open an issue on the GitHub repository.