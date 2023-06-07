import { TextEncoder } from "k6/x/encoding";

export default function () {
  const encoder = new TextEncoder("windows-1252");
  const view = encoder.encode("Hello World");

  console.log(view); // Outputs: [72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100]
}
