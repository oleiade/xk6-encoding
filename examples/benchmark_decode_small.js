import { TextDecoder } from "k6/x/encoding";

export let options = {
  vus: 1,
  duration: "2m",
};

const decoder = new TextDecoder();

export default function () {
  const encoded = new Uint8Array([
    72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100,
  ]);

  decoder.decode(encoded);

  //   console.log(decoded); // Outputs: Hello World

  //   const encoder = new TextEncoder("windows-1252");
  //   const view = encoder.encode("Hello World");
  //   console.log(view);

  //   console.log(`decoder.fatal: ${decoder.fatal}`);
}
