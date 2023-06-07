import { TextDecoder, TextEncoder } from "k6/x/encoding";

export default function () {
  const decoder = new TextDecoder();

  // Suppose we have a string split into two chunks:
  let chunk1 = new Uint8Array([72, 101, 108, 108]); // Decodes to "Hell"
  let chunk2 = new Uint8Array([111, 33]); // Decodes to "o!"

  // We can decode these two chunks in sequence using the stream option:
  let decoded = "";
  decoded += decoder.decode(chunk1, { stream: true }); // Logs: "Hell"
  console.log("decoded: ", decoded);
  decoded += decoder.decode(chunk2, { stream: false }); // Logs: "o!"
  console.log("decoded: ", decoded);
}
