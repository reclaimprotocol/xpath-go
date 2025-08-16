const { JSDOM } = require('jsdom');

const html = "<html><body><input type='text' maxlength='10'/><input type='password' maxlength='20'/><input type='email'/></body></html>";
const xpath = "//input[@maxlength and string-length(@maxlength)=2]";

const dom = new JSDOM(html);
const document = dom.window.document;
const window = dom.window;

const xpathResult = document.evaluate(
    xpath,
    document,
    null,
    window.XPathResult.ORDERED_NODE_ITERATOR_TYPE,
    null
);

console.log('JavaScript XPath results:');
let node;
let count = 0;
while (node = xpathResult.iterateNext()) {
    count++;
    const maxlength = node.getAttribute('maxlength');
    console.log(`${count}: ${node.nodeName.toLowerCase()} - maxlength="${maxlength}" (length: ${maxlength ? maxlength.length : 0})`);
}
console.log(`Total: ${count} results`);

// Test components separately
console.log('\nTesting //input[@maxlength]:');
const maxlengthTest = document.evaluate("//input[@maxlength]", document, null, window.XPathResult.ORDERED_NODE_ITERATOR_TYPE, null);
let maxlengthCount = 0;
while (node = maxlengthTest.iterateNext()) {
    maxlengthCount++;
    const maxlength = node.getAttribute('maxlength');
    console.log(`${maxlengthCount}: maxlength="${maxlength}" (length: ${maxlength.length})`);
}