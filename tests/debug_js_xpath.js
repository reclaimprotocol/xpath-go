// Debug JavaScript XPath evaluation
const { JSDOM } = require('jsdom');

const html = "<html><body><div>Hello</div><p>World</p></body></html>";
const xpath = "//div";

console.log('Testing JavaScript XPath evaluation:');
console.log('HTML:', html);
console.log('XPath:', xpath);

try {
    const dom = new JSDOM(html);
    const document = dom.window.document;
    
    console.log('Document created successfully');
    console.log('Document.evaluate available:', typeof document.evaluate);
    
    // Try to evaluate XPath
    const xpathResult = document.evaluate(
        xpath,
        document,
        null,
        0, // XPathResult.ANY_TYPE
        null
    );
    
    console.log('XPath evaluation successful');
    console.log('Result type:', xpathResult.resultType);
    console.log('Available XPathResult types:', Object.keys(dom.window.XPathResult || {}));
    
} catch (error) {
    console.error('XPath evaluation failed:', error.message);
    console.error('Error details:', error);
}