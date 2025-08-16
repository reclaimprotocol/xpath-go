// Verify XPath contains() behavior with JavaScript's xpath library using jsdom
const { JSDOM } = require('jsdom');
const xpath = require('xpath');

const html = `<html><body>
    <div class="primary active large">Item 1</div>
    <div class="primary inactive">Item 2</div>
    <div class="secondary active">Item 3</div>
</body></html>`;

const dom = new JSDOM(html);
const doc = dom.window.document;

console.log("=== JavaScript XPath contains() Behavior ===");
console.log();

const tests = [
    "//div[contains(@class, 'active')]",
    "//div[contains(@class, 'inactive')]", 
    "//div[contains(@class, 'primary') and contains(@class, 'active')]"
];

tests.forEach(query => {
    try {
        const results = xpath.select(query, doc);
        console.log(`Query: ${query}`);
        console.log(`Results: ${results.length}`);
        results.forEach((node, i) => {
            const textContent = node.textContent || '';
            const className = node.getAttribute ? node.getAttribute('class') : '';
            console.log(`  ${i+1}. "${textContent.trim()}" (class="${className}")`);
        });
        console.log();
    } catch (error) {
        console.log(`Query: ${query}`);
        console.log(`Error: ${error.message}`);
        console.log();
    }
});

console.log("Expected behavior:");
console.log("- contains(@class, 'active') should match 'inactive' because 'inactive' contains 'active'");
console.log("- This is correct XPath substring behavior, not CSS class matching");