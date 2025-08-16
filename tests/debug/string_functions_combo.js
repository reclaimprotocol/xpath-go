const { JSDOM } = require('jsdom');

const html = '<html><body><p>  Hello World  </p><p>Short</p><p>  Very long content here  </p></body></html>';
const xpath = "//p[string-length(normalize-space(text())) > 10]";

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
    console.log(`${count}: ${node.nodeName} - "${node.textContent}" (normalized: "${node.textContent.trim().replace(/\s+/g, ' ')}" - length: ${node.textContent.trim().replace(/\s+/g, ' ').length})`);
}