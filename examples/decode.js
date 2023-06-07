import { TextDecoder, TextEncoder } from "k6/x/encoding";

export default function () {
  const decoder = new TextDecoder();

  const encoded = new Uint8Array([
    72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100,
  ]);

  const decoded = decoder.decode(encoded);

  console.log(decoded); // Outputs: Hello World

  const encoder = new TextEncoder("windows-1252");
  const view = encoder.encode("Hello World");
  console.log(view);
}
