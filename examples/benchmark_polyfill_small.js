// import { TextDecoder, TextEncoder } from "k6/x/encoding";

export let options = {
  vus: 1,
  duration: "2m",
};

function TextEncoder() {}

TextEncoder.prototype.encode = function (string) {
  var octets = [];
  var length = string.length;
  var i = 0;
  while (i < length) {
    var codePoint = string.codePointAt(i);
    var c = 0;
    var bits = 0;
    if (codePoint <= 0x0000007f) {
      c = 0;
      bits = 0x00;
    } else if (codePoint <= 0x000007ff) {
      c = 6;
      bits = 0xc0;
    } else if (codePoint <= 0x0000ffff) {
      c = 12;
      bits = 0xe0;
    } else if (codePoint <= 0x001fffff) {
      c = 18;
      bits = 0xf0;
    }
    octets.push(bits | (codePoint >> c));
    c -= 6;
    while (c >= 0) {
      octets.push(0x80 | ((codePoint >> c) & 0x3f));
      c -= 6;
    }
    i += codePoint >= 0x10000 ? 2 : 1;
  }
  return octets;
};

function TextDecoder() {}

TextDecoder.prototype.decode = function (octets) {
  var string = "";
  var i = 0;
  while (i < octets.length) {
    var octet = octets[i];
    var bytesNeeded = 0;
    var codePoint = 0;
    if (octet <= 0x7f) {
      bytesNeeded = 0;
      codePoint = octet & 0xff;
    } else if (octet <= 0xdf) {
      bytesNeeded = 1;
      codePoint = octet & 0x1f;
    } else if (octet <= 0xef) {
      bytesNeeded = 2;
      codePoint = octet & 0x0f;
    } else if (octet <= 0xf4) {
      bytesNeeded = 3;
      codePoint = octet & 0x07;
    }
    if (octets.length - i - bytesNeeded > 0) {
      var k = 0;
      while (k < bytesNeeded) {
        octet = octets[i + k + 1];
        codePoint = (codePoint << 6) | (octet & 0x3f);
        k += 1;
      }
    } else {
      codePoint = 0xfffd;
      bytesNeeded = octets.length - i;
    }
    string += String.fromCodePoint(codePoint);
    i += bytesNeeded + 1;
  }
  return string;
};

const decoder = new TextDecoder();

export default function () {
  const encoded = new Uint8Array([
    72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100,
  ]);

  decoder.decode(encoded);
}
