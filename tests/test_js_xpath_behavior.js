const { JSDOM } = require('jsdom');

// Test how JavaScript XPath actually behaves with contains() and class attributes
const html = `<html><body>
    <div class='primary active large'>Item 1</div>
    <div class='secondary active'>Item 2</div>
    <div class='primary inactive'>Item 3</div>
</body></html>`;

const dom = new JSDOM(html);
const document = dom.window.document;

console.log('=== Testing JavaScript XPath contains() behavior ===');
console.log('HTML elements:');
console.log('- Item 1: class="primary active large"');
console.log('- Item 2: class="secondary active"');  
console.log('- Item 3: class="primary inactive"');
console.log();

// Test cases to understand JavaScript XPath behavior
const testCases = [
    {
        query: "//div[contains(@class, 'active')]",
        description: "contains(@class, 'active') - should 'active' match 'inactive'?",
    },
    {
        query: "//div[contains(@class, 'inactive')]", 
        description: "contains(@class, 'inactive') - baseline test",
    },
    {
        query: "//div[contains(concat(' ', normalize-space(@class), ' '), ' active ')]",
        description: "Proper whole-word matching for 'active'",
    }
];

testCases.forEach((test, i) => {
    console.log(`${i + 1}. ${test.description}`);
    console.log(`   Query: ${test.query}`);
    
    try {
        const result = document.evaluate(
            test.query,
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        console.log(`   Results: ${result.snapshotLength}`);
        for (let j = 0; j < result.snapshotLength; j++) {
            const node = result.snapshotItem(j);
            console.log(`     ${j + 1}. ${node.textContent.trim()} (class="${node.getAttribute('class')}")`);
        }
    } catch (error) {
        console.log(`   ERROR: ${error.message}`);
    }
    console.log();
});

console.log('=== Key Question ===');
console.log('Does JavaScript XPath contains(@class, "active") match class="primary inactive"?');
console.log('If YES, then contains() does substring matching (my fix was wrong)');
console.log('If NO, then JavaScript does whole-word matching (my fix was correct)');