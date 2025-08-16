const { JSDOM } = require('jsdom');

const html = '<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>';
const xpath = "//div[normalize-space(text())='' and not(*)]";

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
    console.log(`${count}: div - text="${node.textContent}" - normalized="${node.textContent.trim().replace(/\s+/g, ' ')}" - children: ${node.children.length}`);
}
console.log(`Total: ${count} results`);

// Test components separately
console.log('\nTesting //div[normalize-space(text())=\'\']:');
const normalizedTest = document.evaluate("//div[normalize-space(text())='']", document, null, window.XPathResult.ORDERED_NODE_ITERATOR_TYPE, null);
let normalizedCount = 0;
while (node = normalizedTest.iterateNext()) {
    normalizedCount++;
    console.log(`${normalizedCount}: normalized empty - text="${node.textContent}" - children: ${node.children.length}`);
}

console.log('\nTesting //div[not(*)]:');
const notAnyTest = document.evaluate("//div[not(*)]", document, null, window.XPathResult.ORDERED_NODE_ITERATOR_TYPE, null);
let notAnyCount = 0;
while (node = notAnyTest.iterateNext()) {
    notAnyCount++;
    console.log(`${notAnyCount}: no children - text="${node.textContent}" - children: ${node.children.length}`);
}