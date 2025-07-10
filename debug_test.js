// Debug test for the failing case
const { TextDecoder } = require('./encoding');

console.log('Creating decoder...');
const decoder = new TextDecoder();

console.log('First decode - empty case:');
const result1 = decoder.decode(undefined);
console.log('Result 1:', JSON.stringify(result1), 'Length:', result1.length);

console.log('Second decode - streaming with incomplete sequence:');
const result2 = decoder.decode(new Uint8Array([0xc9]), {stream: true});
console.log('Result 2:', JSON.stringify(result2), 'Length:', result2.length);

console.log('Third decode - flush with undefined:');
const result3 = decoder.decode(undefined);
console.log('Result 3:', JSON.stringify(result3), 'Length:', result3.length);
console.log('Result 3 should be replacement character:', result3 === '\uFFFD');