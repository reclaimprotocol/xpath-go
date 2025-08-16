const { JSDOM } = require('jsdom');

const html = '<html><body><section><h1>Title</h1><p>Para</p></section><article><h2>Subtitle</h2><div>Content</div></article></body></html>';
const xpath = "(//section/h1 | //section/p) | (//article/h2 | //article/div)";

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
    console.log(`${count}: ${node.nodeName.toLowerCase()} - "${node.textContent}"`);
}