import { TextEncoder } from "k6/x/encoding";

export default function () {
  const encoder = new TextEncoder("windows-1252");
  const view = encoder.encode("Hello World");
  console.log(view);
}
