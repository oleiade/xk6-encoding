// Test the exact failing scenario from the test
const decoder = new TextDecoder();

console.log("=== Testing the failing case ===");

// Step 1: Decode [0xF0, 0x9F] with stream: true - should return ""
console.log("Step 1: decoder.decode(new Uint8Array([0xF0, 0x9F]), { stream: true })");
const result1 = decoder.decode(new Uint8Array([0xF0, 0x9F]), { stream: true });
console.log("Result 1:", JSON.stringify(result1), "Length:", result1.length);

// Step 2: Decode [0x41] with stream: true - should return "\uFFFDA"
console.log("\nStep 2: decoder.decode(new Uint8Array([0x41]), { stream: true })");
const result2 = decoder.decode(new Uint8Array([0x41]), { stream: true });
console.log("Result 2:", JSON.stringify(result2), "Length:", result2.length);

// Check result
const expected = "\uFFFDA";
console.log("Expected:", JSON.stringify(expected), "Length:", expected.length);
console.log("Match:", result2 === expected);

// Step 3: Final flush
console.log("\nStep 3: decoder.decode()");
const result3 = decoder.decode();
console.log("Result 3:", JSON.stringify(result3), "Length:", result3.length);