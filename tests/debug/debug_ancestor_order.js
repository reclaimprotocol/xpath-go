const { JSDOM } = require('jsdom');

const html = "<html><body><article><section><p id='target'>Content</p></section></article></body></html>";
const xpath = "//p[@id='target']/ancestor-or-self::*";

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

const results = [];
let node;
while (node = xpathResult.iterateNext()) {
    results.push({
        nodeName: node.nodeName.toLowerCase(),
        textContent: node.textContent
    });
}

console.log('JavaScript XPath ancestor-or-self order:');
results.forEach((result, index) => {
    console.log(`${index}: ${result.nodeName} (${result.textContent})`);
});