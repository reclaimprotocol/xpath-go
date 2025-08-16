const { JSDOM } = require('jsdom');

// Test how JavaScript XPath handles ancestor axis with predicates
const html = `<html><body>
    <div class="content">
        <p>Target paragraph</p>
    </div>
</body></html>`;

const dom = new JSDOM(html);
const document = dom.window.document;

console.log('=== Testing JavaScript Ancestor Axis with Predicates ===');

const testCases = [
    "//div[@class='content']",
    "//p",
    "//p[ancestor::div]",
    "//p[ancestor::div[@class='content']]"
];

testCases.forEach((query, i) => {
    console.log(`${i + 1}. Query: ${query}`);
    
    try {
        const result = document.evaluate(
            query,
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        console.log(`   Results: ${result.snapshotLength}`);
        for (let j = 0; j < result.snapshotLength; j++) {
            const node = result.snapshotItem(j);
            console.log(`     ${j + 1}. ${node.textContent.trim()} (tag: ${node.tagName || node.nodeName})`);
        }
    } catch (error) {
        console.log(`   ERROR: ${error.message}`);
    }
    console.log();
});