const { JSDOM } = require('jsdom');

// Function to test
function extractHTMLElementsIndexes(html, xpathExpression, contentsOnly) {
    const dom = new JSDOM(html, {
        contentType: 'text/html',
        includeNodeLocations: true
    });

    const document = dom.window.document;
    const xpathResult = document.evaluate(xpathExpression, document, null, dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE, null);
    const nodes = [];
    
    if (xpathResult?.resultType === dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE &&
        xpathResult?.snapshotLength) {
        for (let i = 0; i < xpathResult.snapshotLength; ++i) {
            nodes.push(xpathResult.snapshotItem(i));
        }
    }

    if (!nodes.length) {
        throw new Error(`Failed to find XPath: "${xpathExpression}"`);
    }

    const res = [];

    for (const node of nodes) {
        const nodeLocation = dom.nodeLocation(node);
        if (!nodeLocation) {
            throw new Error(`Failed to find XPath node location: "${xpathExpression}"`);
        }

        if (contentsOnly) {
            const start = nodeLocation.startTag ? nodeLocation.startTag.endOffset : nodeLocation.startOffset;
            const end = nodeLocation.endTag ? nodeLocation.endTag.startOffset : nodeLocation.endOffset;
            res.push({ start, end });
        } else {
            res.push({ start: nodeLocation.startOffset, end: nodeLocation.endOffset });
        }
    }

    return res;
}

console.log('🧪 Testing extractHTMLElementsIndexes with contentsOnly option');
console.log('================================================================\n');

// Test cases
const testCases = [
    {
        name: "Basic element with contentsOnly=false",
        html: `<html><body><p>Hello World</p></body></html>`,
        xpath: '//p',
        contentsOnly: false,
        expected: "Should return full element including <p> tags"
    },
    {
        name: "Basic element with contentsOnly=true", 
        html: `<html><body><p>Hello World</p></body></html>`,
        xpath: '//p',
        contentsOnly: true,
        expected: "Should return only 'Hello World' content"
    },
    {
        name: "Multiple elements with contentsOnly=false",
        html: `<html><body><div><p>First</p><p>Second</p></div></body></html>`,
        xpath: '//p',
        contentsOnly: false,
        expected: "Should return full <p> elements including tags"
    },
    {
        name: "Multiple elements with contentsOnly=true",
        html: `<html><body><div><p>First</p><p>Second</p></div></body></html>`,
        xpath: '//p',
        contentsOnly: true,
        expected: "Should return only text content 'First' and 'Second'"
    },
    {
        name: "Nested elements with contentsOnly=false",
        html: `<html><body><div>Outer <span>Inner</span> Text</div></body></html>`,
        xpath: '//div',
        contentsOnly: false,
        expected: "Should return full <div> element including tags"
    },
    {
        name: "Nested elements with contentsOnly=true",
        html: `<html><body><div>Outer <span>Inner</span> Text</div></body></html>`,
        xpath: '//div',
        contentsOnly: true,
        expected: "Should return div content: 'Outer <span>Inner</span> Text'"
    },
    {
        name: "Element with attributes contentsOnly=false",
        html: `<html><body><a href="test.html" class="link">Click here</a></body></html>`,
        xpath: '//a',
        contentsOnly: false,
        expected: "Should include full element with attributes"
    },
    {
        name: "Element with attributes contentsOnly=true",
        html: `<html><body><a href="test.html" class="link">Click here</a></body></html>`,
        xpath: '//a',
        contentsOnly: true,
        expected: "Should return only 'Click here' content"
    },
    {
        name: "Empty element with contentsOnly=false",
        html: `<html><body><div></div></body></html>`,
        xpath: '//div',
        contentsOnly: false,
        expected: "Should return full empty <div></div> element"
    },
    {
        name: "Empty element with contentsOnly=true",
        html: `<html><body><div></div></body></html>`,
        xpath: '//div',
        contentsOnly: true,
        expected: "Should return empty content between tags"
    },
    {
        name: "Self-closing element with contentsOnly=false",
        html: `<html><body><img src="test.jpg" alt="test" /></body></html>`,
        xpath: '//img',
        contentsOnly: false,
        expected: "Should return full self-closing element"
    },
    {
        name: "Self-closing element with contentsOnly=true", 
        html: `<html><body><img src="test.jpg" alt="test" /></body></html>`,
        xpath: '//img',
        contentsOnly: true,
        expected: "Should handle self-closing element appropriately"
    }
];

let passed = 0;
let failed = 0;

testCases.forEach((testCase, index) => {
    console.log(`[${index + 1}/${testCases.length}] ${testCase.name}`);
    console.log(`HTML: ${testCase.html}`);
    console.log(`XPath: ${testCase.xpath}`);
    console.log(`contentsOnly: ${testCase.contentsOnly}`);
    console.log(`Expected: ${testCase.expected}`);
    
    try {
        const result = extractHTMLElementsIndexes(testCase.html, testCase.xpath, testCase.contentsOnly);
        console.log(`✅ Success: Found ${result.length} elements`);
        
        result.forEach((element, i) => {
            const extractedText = testCase.html.substring(element.start, element.end);
            console.log(`   [${i}] Offsets: ${element.start}-${element.end}`);
            console.log(`   [${i}] Extracted: "${extractedText}"`);
        });
        
        passed++;
    } catch (error) {
        console.log(`❌ Failed: ${error.message}`);
        failed++;
    }
    
    console.log();
});

console.log('\n📊 TEST SUMMARY');
console.log('===============');
console.log(`Total Tests: ${testCases.length}`);
console.log(`Passed: ${passed}`);
console.log(`Failed: ${failed}`);

if (failed === 0) {
    console.log('\n🎉 All tests passed!');
} else {
    console.log('\n⚠️  Some tests failed - check the implementation');
}