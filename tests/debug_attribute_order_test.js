const { JSDOM } = require('jsdom');

const html = `<html><body><input type='text' name='username' required /></body></html>`;

console.log('🔍 Testing attribute order in different scenarios...\n');

// Test 1: Original order
const dom1 = new JSDOM(html);
const doc1 = dom1.window.document;
const result1 = doc1.evaluate('//input/@*', doc1, null, 0, null);
let node1 = result1.iterateNext();
console.log('Original order (type, name, required):');
let index = 0;
while (node1) {
    console.log(`  ${index}: ${node1.nodeName} = "${node1.nodeValue}"`);
    node1 = result1.iterateNext();
    index++;
}

// Test 2: Different order in HTML  
const html2 = `<html><body><input required name='username' type='text' /></body></html>`;
const dom2 = new JSDOM(html2);
const doc2 = dom2.window.document;
const result2 = doc2.evaluate('//input/@*', doc2, null, 0, null);
let node2 = result2.iterateNext();
console.log('\nDifferent order (required, name, type):');
index = 0;
while (node2) {
    console.log(`  ${index}: ${node2.nodeName} = "${node2.nodeValue}"`);
    node2 = result2.iterateNext();
    index++;
}

// Test 3: Programmatically set attributes
const html3 = `<html><body><input /></body></html>`;
const dom3 = new JSDOM(html3);
const doc3 = dom3.window.document;
const input3 = doc3.querySelector('input');
input3.setAttribute('zebra', 'last');
input3.setAttribute('alpha', 'first');
input3.setAttribute('middle', 'between');
const result3 = doc3.evaluate('//input/@*', doc3, null, 0, null);
let node3 = result3.iterateNext();
console.log('\nProgrammatically set (zebra, alpha, middle):');
index = 0;
while (node3) {
    console.log(`  ${index}: ${node3.nodeName} = "${node3.nodeValue}"`);
    node3 = result3.iterateNext();
    index++;
}