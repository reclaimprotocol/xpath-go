#!/usr/bin/env node

const { JSDOM } = require('jsdom');

function testEmptySubstring() {
    const html = `<html><body><p>Mid</p></body></html>`;
    const xpath = `//p[substring(text(), 0) = ""]`;
    
    console.log("=== TESTING SUBSTRING WITH POSITION 0 ===\n");
    console.log(`XPath: ${xpath}`);
    
    const dom = new JSDOM(html);
    const document = dom.window.document;
    
    try {
        const result = document.evaluate(
            xpath,
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        console.log(`JavaScript found: ${result.snapshotLength} results`);
        
        for (let i = 0; i < result.snapshotLength; i++) {
            const node = result.snapshotItem(i);
            console.log(`  Result ${i + 1}: <${node.nodeName.toLowerCase()}>${node.textContent}</${node.nodeName.toLowerCase()}>`);
        }
        
        // Test what substring(text(), 0) actually returns
        console.log("\n--- Testing substring behavior ---");
        const text = "Mid";
        console.log(`Text: "${text}"`);
        console.log(`JavaScript substring(0): "${text.substring(0-1)}"`); // Simulate XPath 1-based to JS 0-based conversion
        
    } catch (error) {
        console.error(`JavaScript XPath error: ${error.message}`);
    }
}

testEmptySubstring();