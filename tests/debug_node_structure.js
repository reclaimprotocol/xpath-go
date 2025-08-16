// Debug the node structure to understand the issue
const jsdom = require('jsdom');
const { JSDOM } = jsdom;

const html = `<html><body><article><header><h1>Title</h1></header><section><p>Content</p><aside><p>Sidebar</p></aside></section></article></body></html>`;
const dom = new JSDOM(html);
const document = dom.window.document;

// Find all p elements in section
const pElements = document.evaluate('//article//section//p', document, null, 0, null);
console.log('All p elements in section:');
let node;
while (node = pElements.iterateNext()) {
  console.log(`- Text: "${node.textContent}"`);
  
  // Check ancestors
  let current = node.parentNode;
  console.log('  Ancestors:');
  while (current && current.nodeName !== '#document') {
    console.log(`    ${current.nodeName.toLowerCase()}`);
    current = current.parentNode;
  }
  
  // Test ancestor::aside specifically
  const hasAsideAncestor = document.evaluate('ancestor::aside', node, null, 0, null).iterateNext() !== null;
  console.log(`  Has aside ancestor: ${hasAsideAncestor}`);
  console.log();
}

// Now test the full XPath
const filteredElements = document.evaluate('//article//section//p[not(ancestor::aside)]', document, null, 0, null);
console.log('P elements NOT in aside:');
while (node = filteredElements.iterateNext()) {
  console.log(`- Text: "${node.textContent}"`);
}