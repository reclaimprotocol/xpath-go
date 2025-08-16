const { JSDOM } = require('jsdom');

const html = '<html><body><ul><li><span>Item 1</span><!-- comment --></li><li>Item 2</li><li><a href="#">Item 3</a><span>Extra</span></li></ul></body></html>';
const xpath = "//li[span and not(a)]";

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
    console.log(`${count}: ${node.nodeName.toLowerCase()} - "${node.textContent}" - has span: ${!!node.querySelector('span')} - has a: ${!!node.querySelector('a')}`);
}
console.log(`Total: ${count} results`);

// Also test the components separately
console.log('\nTesting //li[span]:');
const spanTest = document.evaluate("//li[span]", document, null, window.XPathResult.ORDERED_NODE_ITERATOR_TYPE, null);
let spanCount = 0;
while (node = spanTest.iterateNext()) {
    spanCount++;
    console.log(`${spanCount}: has span - "${node.textContent.trim()}"`);
}

console.log('\nTesting //li[not(a)]:');
const notATest = document.evaluate("//li[not(a)]", document, null, window.XPathResult.ORDERED_NODE_ITERATOR_TYPE, null);
let notACount = 0;
while (node = notATest.iterateNext()) {
    notACount++;
    console.log(`${notACount}: no a element - "${node.textContent.trim()}"`);
}