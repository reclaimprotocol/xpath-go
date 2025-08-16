// Debug the attribute order in JavaScript
const jsdom = require('jsdom');
const { JSDOM } = jsdom;

const html = `<html><body><input type='text' name='username' required /></body></html>`;
const dom = new JSDOM(html);
const document = dom.window.document;

console.log('JavaScript attribute order:');

// Test XPath attribute selection
const result = document.evaluate('//input/@*', document, null, 0, null);
let node;
let index = 0;
while (node = result.iterateNext()) {
  console.log(`${index}: ${node.nodeName} = "${node.nodeValue}"`);
  index++;
}

console.log('\nDirect attribute inspection:');
const input = document.querySelector('input');
console.log('Input attributes:', input.getAttributeNames());
for (let attr of input.attributes) {
  console.log(`- ${attr.name} = "${attr.value}"`);
}